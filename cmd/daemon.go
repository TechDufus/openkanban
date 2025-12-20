package cmd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/techdufus/openkanban/internal/daemon"
	"github.com/techdufus/openkanban/internal/daemon/service"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the OpenKanban daemon",
	Long:  `The daemon manages persistent PTY sessions for AI agents.`,
}

var daemonRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run daemon in foreground",
	RunE: func(cmd *cobra.Command, args []string) error {
		socketPath := defaultSocketPath()
		log.Printf("starting daemon on %s", socketPath)

		server := daemon.NewServer(socketPath)
		if err := server.Start(); err != nil {
			return fmt.Errorf("failed to start: %w", err)
		}

		go func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			<-sigCh
			log.Println("shutting down...")
			server.Close()
			os.Exit(0)
		}()

		log.Println("listening for connections...")
		return server.Accept()
	},
}

var daemonInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install daemon as system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		binPath, err := service.GetBinaryPath()
		if err != nil {
			return fmt.Errorf("getting binary path: %w", err)
		}

		platform := service.DetectPlatform()
		fmt.Printf("Platform: %s\n", platform)
		fmt.Printf("Binary: %s\n", binPath)

		if err := mgr.Install(binPath); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}

		fmt.Println("Service installed.")
		fmt.Println("Run 'openkanban daemon start' to start the daemon.")
		return nil
	},
}

var daemonUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove daemon system service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		if err := mgr.Uninstall(); err != nil {
			return fmt.Errorf("uninstall failed: %w", err)
		}

		fmt.Println("Service uninstalled.")
		return nil
	},
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the daemon service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		if !mgr.IsInstalled() {
			return fmt.Errorf("service not installed. Run 'openkanban daemon install' first")
		}

		if err := mgr.Start(); err != nil {
			return fmt.Errorf("start failed: %w", err)
		}

		fmt.Println("Service started.")
		return nil
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the daemon service",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		if err := mgr.Stop(); err != nil {
			return fmt.Errorf("stop failed: %w", err)
		}

		fmt.Println("Service stopped.")
		return nil
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := service.NewManager()
		if err != nil {
			return err
		}

		platform := service.DetectPlatform()
		fmt.Printf("Platform: %s\n", platform)
		fmt.Printf("Installed: %v\n", mgr.IsInstalled())

		status, err := mgr.Status()
		if err != nil {
			fmt.Printf("Status: error (%v)\n", err)
		} else {
			fmt.Printf("Status: %s\n", status)
		}

		sockPath := defaultSocketPath()
		if _, err := os.Stat(sockPath); err == nil {
			fmt.Printf("Socket: %s (exists)\n", sockPath)
		} else {
			fmt.Printf("Socket: %s (not found)\n", sockPath)
		}

		return nil
	},
}

func init() {
	daemonCmd.AddCommand(daemonRunCmd)
	daemonCmd.AddCommand(daemonInstallCmd)
	daemonCmd.AddCommand(daemonUninstallCmd)
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)

	rootCmd.AddCommand(daemonCmd)
}

func defaultSocketPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".openkanban", "daemon.sock")
}

func EnsureDaemon() error {
	sockPath := defaultSocketPath()

	if _, err := os.Stat(sockPath); err == nil {
		client := daemon.NewClient(sockPath)
		if err := client.Connect(); err == nil {
			client.Close()
			return nil
		}
		os.Remove(sockPath)
	}

	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable: %w", err)
	}

	logDir := filepath.Join(filepath.Dir(sockPath))
	os.MkdirAll(logDir, 0755)

	logFile := filepath.Join(logDir, "daemon.log")

	cmd := exec.Command(binPath, "daemon", "run")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644); err == nil {
		cmd.Stdout = f
		cmd.Stderr = f
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting daemon: %w", err)
	}

	for i := 0; i < 50; i++ {
		time.Sleep(100 * time.Millisecond)
		if _, err := os.Stat(sockPath); err == nil {
			client := daemon.NewClient(sockPath)
			if err := client.Connect(); err == nil {
				client.Close()
				return nil
			}
		}
	}

	return fmt.Errorf("daemon failed to start within 5 seconds")
}
