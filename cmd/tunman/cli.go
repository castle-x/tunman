package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/yourusername/tunman/internal/core"
	"github.com/yourusername/tunman/internal/i18n"
	"github.com/yourusername/tunman/internal/model"
)

type tunnelStorage interface {
	LoadTunnels() ([]model.Tunnel, error)
	AddTunnel(tunnel model.Tunnel) error
	UpdateTunnel(tunnel model.Tunnel) error
	DeleteTunnel(id string) error
	ReadLogs(id string, lines int) (string, error)
	LogPath(id string) string
}

type tunnelController interface {
	CheckCloudflared() error
	CheckAuth() error
	SetupTunnel(tunnel *model.Tunnel) error
	TeardownTunnel(tunnel *model.Tunnel) error
	Start(tunnel *model.Tunnel) error
	Stop(tunnel *model.Tunnel) error
	Restart(tunnel *model.Tunnel) error
	SyncStatus(tunnels []model.Tunnel) []model.Tunnel
	EditTunnel(tunnel *model.Tunnel) error
	WriteConfigYML(tunnel *model.Tunnel) error
}

type cliApp struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	storage    tunnelStorage
	controller tunnelController
	version    string
	buildTime  string
}

type usageError struct {
	message string
}

func (e usageError) Error() string {
	return e.message
}

type createOptions struct {
	category    string
	baseDomain  string
	prefix      string
	description string
	port        int
}

func newCLIApp(stdout, stderr io.Writer) *cliApp {
	storage := core.NewStorage()
	return &cliApp{
		stdin:      os.Stdin,
		stdout:     stdout,
		stderr:     stderr,
		storage:    storage,
		controller: core.NewController(storage),
		version:    Version,
		buildTime:  BuildTime,
	}
}

func runCLI(args []string) int {
	return newCLIApp(os.Stdout, os.Stderr).Run(args)
}

func (a *cliApp) Run(args []string) int {
	if len(args) == 0 {
		a.printUsage(a.stderr)
		return 1
	}

	command := args[0]
	if command == "-h" || command == "--help" {
		command = "help"
	}

	var err error
	switch command {
	case "help":
		err = a.cmdHelp(args[1:])
	case "version", "v":
		err = a.cmdVersion(args[1:])
	case "list", "ls":
		err = a.cmdList(args[1:])
	case "status":
		err = a.cmdStatus(args[1:])
	case "start":
		err = a.cmdStart(args[1:])
	case "stop":
		err = a.cmdStop(args[1:])
	case "restart":
		err = a.cmdRestart(args[1:])
	case "logs", "log":
		err = a.cmdLogs(args[1:])
	case "delete", "rm":
		err = a.cmdDelete(args[1:])
	case "edit":
		err = a.cmdEdit(args[1:])
	case "create", "new":
		err = a.cmdCreate(args[1:])
	case "temp":
		err = a.cmdTemp(args[1:])
	default:
		fmt.Fprintln(a.stderr, i18n.Tf("err_cmd_unknown", args[0]))
		a.printUsage(a.stderr)
		return 1
	}

	if err == nil {
		return 0
	}

	fmt.Fprintln(a.stderr, err.Error())
	var usageErr usageError
	if errors.As(err, &usageErr) {
		return 2
	}
	return 1
}

func (a *cliApp) cmdHelp(args []string) error {
	if len(args) > 0 {
		return usageError{message: i18n.T("cmd_help_usage")}
	}
	a.printUsage(a.stdout)
	return nil
}

func (a *cliApp) cmdVersion(args []string) error {
	if len(args) > 0 {
		return usageError{message: i18n.T("cmd_version_usage")}
	}

	fmt.Fprintf(a.stdout, "TunMan %s", a.version)
	if a.buildTime != "" {
		fmt.Fprintf(a.stdout, " (%s)", a.buildTime)
	}
	fmt.Fprintln(a.stdout)
	return nil
}

func (a *cliApp) cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	categoryArg := fs.String("category", "", "")
	statusArg := fs.String("status", "", "")
	if err := fs.Parse(args); err != nil {
		return usageError{message: i18n.T("cmd_list_usage")}
	}
	if fs.NArg() != 0 {
		return usageError{message: i18n.T("cmd_list_usage")}
	}

	tunnels, err := a.loadSyncedTunnels()
	if err != nil {
		return err
	}

	filtered, err := filterTunnels(tunnels, *categoryArg, *statusArg)
	if err != nil {
		return err
	}
	if len(filtered) == 0 {
		fmt.Fprintln(a.stdout, i18n.T("cmd_list_empty"))
		return nil
	}

	renderTunnelTable(a.stdout, filtered)
	fmt.Fprintln(a.stdout, i18n.Tf("cmd_list_summary", len(filtered), countByStatus(filtered, model.StatusRunning), countByStatus(filtered, model.StatusStopped), countByStatus(filtered, model.StatusError)))
	return nil
}

func (a *cliApp) cmdStatus(args []string) error {
	switch len(args) {
	case 0:
		return a.cmdList(nil)
	case 1:
		tunnel, err := a.getTunnel(args[0])
		if err != nil {
			return err
		}
		printTunnelDetails(a.stdout, *tunnel)
		return nil
	default:
		return usageError{message: i18n.T("cmd_status_usage")}
	}
}

func (a *cliApp) cmdStart(args []string) error {
	tunnel, err := a.loadTunnelForAction(args, i18n.T("cmd_start_usage"))
	if err != nil {
		return err
	}
	if err := ensureRunnableTunnel(a.controller, tunnel); err != nil {
		return err
	}
	if err := a.controller.Start(tunnel); err != nil {
		return err
	}
	if err := a.storage.UpdateTunnel(*tunnel); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, i18n.Tf("cmd_started", tunnel.ID, tunnel.Name))
	return nil
}

func (a *cliApp) cmdStop(args []string) error {
	tunnel, err := a.loadTunnelForAction(args, i18n.T("cmd_stop_usage"))
	if err != nil {
		return err
	}
	if err := a.controller.Stop(tunnel); err != nil {
		return err
	}
	if err := a.storage.UpdateTunnel(*tunnel); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, i18n.Tf("cmd_stopped", tunnel.ID, tunnel.Name))
	return nil
}

func (a *cliApp) cmdRestart(args []string) error {
	tunnel, err := a.loadTunnelForAction(args, i18n.T("cmd_restart_usage"))
	if err != nil {
		return err
	}
	if err := ensureRunnableTunnel(a.controller, tunnel); err != nil {
		return err
	}
	if err := a.controller.Restart(tunnel); err != nil {
		return err
	}
	if err := a.storage.UpdateTunnel(*tunnel); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, i18n.Tf("cmd_restarted", tunnel.ID, tunnel.Name))
	return nil
}

func (a *cliApp) cmdLogs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	follow := fs.Bool("follow", false, "")
	fs.BoolVar(follow, "f", false, "")
	lines := fs.Int("lines", 100, "")
	if err := fs.Parse(args); err != nil {
		return usageError{message: i18n.T("cmd_logs_usage")}
	}
	if fs.NArg() != 1 {
		return usageError{message: i18n.T("cmd_logs_usage")}
	}

	tunnel, err := a.getTunnel(fs.Arg(0))
	if err != nil {
		return err
	}

	if *follow {
		logPath := a.storage.LogPath(tunnel.ID)
		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_RDONLY, 0644)
		if err == nil {
			_ = file.Close()
		}
		cmd := exec.Command("tail", "-n", fmt.Sprintf("%d", *lines), "-f", logPath)
		cmd.Stdout = a.stdout
		cmd.Stderr = a.stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}

	logs, err := a.storage.ReadLogs(tunnel.ID, *lines)
	if err != nil {
		return err
	}
	if strings.TrimSpace(logs) == "" {
		fmt.Fprintln(a.stdout, i18n.T("cmd_logs_empty"))
		return nil
	}
	fmt.Fprint(a.stdout, logs)
	if !strings.HasSuffix(logs, "\n") {
		fmt.Fprintln(a.stdout)
	}
	return nil
}

func (a *cliApp) cmdDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	yes := fs.Bool("yes", false, "")
	fs.BoolVar(yes, "y", false, "")
	if err := fs.Parse(args); err != nil {
		return usageError{message: i18n.T("cmd_delete_usage")}
	}
	if fs.NArg() != 1 {
		return usageError{message: i18n.T("cmd_delete_usage")}
	}

	tunnel, err := a.getTunnel(fs.Arg(0))
	if err != nil {
		return err
	}

	if !*yes {
		confirmed, err := a.confirmDelete(tunnel)
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintln(a.stdout, i18n.T("cmd_delete_canceled"))
			return nil
		}
	}

	if tunnel.Status == model.StatusRunning {
		if err := a.controller.Stop(tunnel); err != nil {
			return err
		}
	}
	if err := a.controller.TeardownTunnel(tunnel); err != nil {
		return err
	}
	if err := a.storage.DeleteTunnel(tunnel.ID); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, i18n.Tf("cmd_deleted", tunnel.ID, tunnel.Name))
	return nil
}

func (a *cliApp) cmdEdit(args []string) error {
	if len(args) != 1 {
		return usageError{message: i18n.T("cmd_edit_usage")}
	}

	tunnel, err := a.getTunnel(args[0])
	if err != nil {
		return err
	}
	if err := a.controller.EditTunnel(tunnel); err != nil {
		return err
	}
	if tunnel.Category != model.CategoryEphemeral && tunnel.TunnelID != "" {
		if err := a.controller.WriteConfigYML(tunnel); err != nil {
			return err
		}
	}
	fmt.Fprintln(a.stdout, i18n.Tf("cmd_edited", tunnel.ID, tunnel.Name))
	return nil
}

func (a *cliApp) cmdCreate(args []string) error {
	opts, err := parseCreateOptions(args)
	if err != nil {
		return err
	}

	category, err := normalizeCategory(opts.category)
	if err != nil {
		return err
	}
	if err := validateCreateOptions(category, opts); err != nil {
		return err
	}

	input := core.CreateTunnelInput{
		Category:    category,
		BaseDomain:  opts.baseDomain,
		Prefix:      opts.prefix,
		Port:        opts.port,
		Description: opts.description,
	}
	tunnel, err := input.Build()
	if err != nil {
		return err
	}

	if tunnel.Category != model.CategoryEphemeral {
		if err := a.controller.CheckCloudflared(); err != nil {
			return fmt.Errorf(i18n.T("create_err_cf_not_installed"))
		}
		if err := a.controller.CheckAuth(); err != nil {
			return fmt.Errorf(i18n.T("create_err_cf_not_logged_in"))
		}
		if err := a.controller.SetupTunnel(&tunnel); err != nil {
			return err
		}
	}

	if err := a.storage.AddTunnel(tunnel); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, i18n.Tf("cmd_created", tunnel.ID, tunnel.Name, tunnel.DisplayURL()))

	if tunnel.Category != model.CategoryEphemeral {
		if err := a.controller.Start(&tunnel); err != nil {
			return err
		}
		if err := a.storage.UpdateTunnel(tunnel); err != nil {
			return err
		}
		fmt.Fprintln(a.stdout, i18n.Tf("cmd_started", tunnel.ID, tunnel.Name))
	}
	return nil
}

func (a *cliApp) cmdTemp(args []string) error {
	var portArg string
	remaining := args
	if len(remaining) > 0 && !strings.HasPrefix(remaining[0], "-") {
		portArg = strings.TrimSpace(remaining[0])
		remaining = remaining[1:]
	}

	fs := flag.NewFlagSet("temp", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	description := fs.String("desc", "", "")
	if err := fs.Parse(remaining); err != nil {
		return usageError{message: i18n.T("cmd_temp_usage")}
	}
	if fs.NArg() != 0 || portArg == "" {
		return usageError{message: i18n.T("cmd_temp_usage")}
	}

	cmdArgs := []string{"ephemeral", "--port", portArg}
	if *description != "" {
		cmdArgs = append(cmdArgs, "--desc", *description)
	}
	return a.cmdCreate(cmdArgs)
}

func (a *cliApp) loadSyncedTunnels() ([]model.Tunnel, error) {
	tunnels, err := a.storage.LoadTunnels()
	if err != nil {
		return nil, err
	}
	tunnels = a.controller.SyncStatus(tunnels)
	sort.SliceStable(tunnels, func(i, j int) bool {
		return tunnels[i].CreatedAt.Time().After(tunnels[j].CreatedAt.Time())
	})
	return tunnels, nil
}

func (a *cliApp) loadTunnelForAction(args []string, usage string) (*model.Tunnel, error) {
	if len(args) != 1 {
		return nil, usageError{message: usage}
	}
	return a.getTunnel(args[0])
}

func (a *cliApp) getTunnel(id string) (*model.Tunnel, error) {
	tunnels, err := a.loadSyncedTunnels()
	if err != nil {
		return nil, err
	}
	for i := range tunnels {
		if tunnels[i].ID == id {
			return &tunnels[i], nil
		}
	}
	return nil, fmt.Errorf(i18n.Tf("cmd_tunnel_not_found", id))
}

func (a *cliApp) confirmDelete(tunnel *model.Tunnel) (bool, error) {
	fmt.Fprintf(a.stdout, i18n.Tf("cmd_delete_confirm", tunnel.ID, tunnel.Name))
	reader := bufio.NewReader(a.stdin)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes", nil
}

func (a *cliApp) printUsage(w io.Writer) {
	fmt.Fprintln(w, i18n.T("cmd_usage"))
	fmt.Fprintln(w)
	for _, line := range []string{
		i18n.T("cmd_help_tui"),
		i18n.T("cmd_help_list"),
		i18n.T("cmd_help_status"),
		i18n.T("cmd_help_start"),
		i18n.T("cmd_help_stop"),
		i18n.T("cmd_help_restart"),
		i18n.T("cmd_help_logs"),
		i18n.T("cmd_help_create"),
		i18n.T("cmd_help_temp"),
		i18n.T("cmd_help_edit"),
		i18n.T("cmd_help_delete"),
		i18n.T("cmd_help_version"),
		i18n.T("cmd_help_help"),
	} {
		fmt.Fprintln(w, line)
	}
}

func parseCreateOptions(args []string) (createOptions, error) {
	var positionalCategory string
	remaining := args
	if len(remaining) > 0 && !strings.HasPrefix(remaining[0], "-") {
		positionalCategory = remaining[0]
		remaining = remaining[1:]
	}

	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var opts createOptions
	fs.StringVar(&opts.category, "category", positionalCategory, "")
	fs.StringVar(&opts.category, "c", positionalCategory, "")
	fs.StringVar(&opts.baseDomain, "domain", "", "")
	fs.StringVar(&opts.baseDomain, "d", "", "")
	fs.StringVar(&opts.prefix, "prefix", "", "")
	fs.StringVar(&opts.prefix, "p", "", "")
	fs.IntVar(&opts.port, "port", 0, "")
	fs.IntVar(&opts.port, "P", 0, "")
	fs.StringVar(&opts.description, "desc", "", "")
	fs.StringVar(&opts.description, "description", "", "")
	if err := fs.Parse(remaining); err != nil {
		return createOptions{}, usageError{message: i18n.T("cmd_create_usage")}
	}
	if fs.NArg() != 0 {
		return createOptions{}, usageError{message: i18n.T("cmd_create_usage")}
	}
	if opts.category == "" {
		opts.category = string(model.CategoryCustom)
	}
	return opts, nil
}

func validateCreateOptions(category model.Category, opts createOptions) error {
	if opts.port <= 0 || opts.port > 65535 {
		return usageError{message: i18n.T("create_err_port_range")}
	}

	switch category {
	case model.CategoryCustom:
		if strings.TrimSpace(opts.baseDomain) == "" {
			return usageError{message: i18n.T("create_err_base_domain_required")}
		}
		if strings.TrimSpace(opts.prefix) == "" {
			return usageError{message: i18n.T("create_err_prefix_required")}
		}
	case model.CategoryTesting:
		if strings.TrimSpace(opts.baseDomain) == "" {
			return usageError{message: i18n.T("create_err_base_domain_required")}
		}
	}
	return nil
}

func normalizeCategory(value string) (model.Category, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return model.CategoryCustom, nil
	}
	category, err := model.ParseCategory(trimmed)
	if err != nil {
		return "", usageError{message: i18n.Tf("cmd_create_invalid_category", value)}
	}
	return category, nil
}

func ensureRunnableTunnel(controller tunnelController, tunnel *model.Tunnel) error {
	if tunnel.Category == model.CategoryEphemeral {
		return nil
	}
	if strings.TrimSpace(tunnel.TunnelID) == "" {
		return fmt.Errorf(i18n.Tf("cmd_tunnel_missing_id", tunnel.ID))
	}
	return controller.WriteConfigYML(tunnel)
}

func filterTunnels(tunnels []model.Tunnel, categoryArg, statusArg string) ([]model.Tunnel, error) {
	var category model.Category
	var status model.Status
	var err error

	if strings.TrimSpace(categoryArg) != "" {
		category, err = model.ParseCategory(categoryArg)
		if err != nil {
			return nil, usageError{message: i18n.Tf("cmd_list_bad_category", categoryArg)}
		}
	}
	if strings.TrimSpace(statusArg) != "" {
		status, err = model.ParseStatus(statusArg)
		if err != nil {
			return nil, usageError{message: i18n.Tf("cmd_list_bad_status", statusArg)}
		}
	}

	filtered := make([]model.Tunnel, 0, len(tunnels))
	for _, tunnel := range tunnels {
		if category != "" && tunnel.Category != category {
			continue
		}
		if status != "" && tunnel.Status != status {
			continue
		}
		filtered = append(filtered, tunnel)
	}
	return filtered, nil
}

func renderTunnelTable(w io.Writer, tunnels []model.Tunnel) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
		i18n.T("cmd_col_id"),
		i18n.T("cmd_col_category"),
		i18n.T("cmd_col_status"),
		i18n.T("cmd_col_name"),
		i18n.T("cmd_col_port"),
		i18n.T("cmd_col_endpoint"),
	)
	for _, tunnel := range tunnels {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\n",
			tunnel.ID,
			categoryLabel(tunnel.Category),
			statusLabel(tunnel.Status),
			tunnel.Name,
			tunnel.Port,
			tunnel.DisplayURL(),
		)
	}
	_ = tw.Flush()
}

func printTunnelDetails(w io.Writer, tunnel model.Tunnel) {
	lines := []string{
		fmt.Sprintf("%s: %s", i18n.T("cmd_field_id"), tunnel.ID),
		fmt.Sprintf("%s: %s", i18n.T("cmd_field_name"), tunnel.Name),
		fmt.Sprintf("%s: %s", i18n.T("cmd_field_category"), categoryLabel(tunnel.Category)),
		fmt.Sprintf("%s: %s", i18n.T("cmd_field_status"), statusLabel(tunnel.Status)),
		fmt.Sprintf("%s: %d", i18n.T("cmd_field_port"), tunnel.Port),
		fmt.Sprintf("%s: %s", i18n.T("cmd_field_endpoint"), tunnel.DisplayURL()),
	}
	if tunnel.Description != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", i18n.T("cmd_field_desc"), tunnel.Description))
	}
	if tunnel.BaseDomain != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", i18n.T("cmd_field_domain"), tunnel.BaseDomain))
	}
	if tunnel.Prefix != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", i18n.T("cmd_field_prefix"), tunnel.Prefix))
	}
	if tunnel.TunnelID != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", i18n.T("cmd_field_tunnel_id"), tunnel.TunnelID))
	}
	if tunnel.SessionName != "" {
		lines = append(lines, fmt.Sprintf("%s: %s", i18n.T("cmd_field_session"), tunnel.SessionName))
	}
	if tunnel.PID > 0 {
		lines = append(lines, fmt.Sprintf("%s: %d", i18n.T("cmd_field_pid"), tunnel.PID))
	}
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
}

func countByStatus(tunnels []model.Tunnel, status model.Status) int {
	count := 0
	for _, tunnel := range tunnels {
		if tunnel.Status == status {
			count++
		}
	}
	return count
}

func categoryLabel(category model.Category) string {
	switch category {
	case model.CategoryCustom:
		return i18n.T("cat_custom")
	case model.CategoryTesting:
		return i18n.T("cat_testing")
	case model.CategoryEphemeral:
		return i18n.T("cat_ephemeral")
	default:
		return string(category)
	}
}

func statusLabel(status model.Status) string {
	switch status {
	case model.StatusRunning:
		return i18n.T("status_running")
	case model.StatusStopped:
		return i18n.T("status_stopped")
	case model.StatusError:
		return i18n.T("status_error")
	default:
		return string(status)
	}
}
