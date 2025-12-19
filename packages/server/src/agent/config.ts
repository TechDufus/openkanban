import type { AgentConfig } from "@openkanban/shared"
import os from "node:os"
import path from "node:path"
import fs from "node:fs"

const OPENCODE_CONFIG: AgentConfig = {
  name: "opencode",
  command: "opencode",
  args: [],
  env: {},
  statusApi: "http://localhost:4096/session/status",
  initPrompt: "{{Title}}\n\n{{Description}}",
}

const DEFAULT_AGENTS: AgentConfig[] = [
  OPENCODE_CONFIG,
  {
    name: "claude",
    command: "claude",
    args: [],
    env: {},
    initPrompt: "{{Title}}\n\n{{Description}}",
  },
  {
    name: "aider",
    command: "aider",
    args: ["--yes"],
    env: {},
    initPrompt: "{{Title}}\n\n{{Description}}",
  },
  {
    name: "bash",
    command: "bash",
    args: [],
    env: {},
  },
  {
    name: "shell",
    command: process.env["SHELL"] ?? "/bin/sh",
    args: [],
    env: {},
  },
]

class AgentConfigManager {
  private configs: Map<string, AgentConfig> = new Map()

  constructor() {
    for (const config of DEFAULT_AGENTS) {
      this.configs.set(config.name, config)
    }
    this.loadUserConfigs()
  }

  private loadUserConfigs(): void {
    const configPath = path.join(os.homedir(), ".config", "openkanban", "agents.json")
    
    if (!fs.existsSync(configPath)) {
      return
    }

    try {
      const content = fs.readFileSync(configPath, "utf-8")
      const userConfigs = JSON.parse(content) as AgentConfig[]
      
      for (const config of userConfigs) {
        if (config.name && config.command) {
          this.configs.set(config.name, {
            ...this.configs.get(config.name),
            ...config,
          })
        }
      }
    } catch {
      return
    }
  }

  get(name: string): AgentConfig | undefined {
    return this.configs.get(name)
  }

  getOrDefault(name: string): AgentConfig {
    return this.configs.get(name) ?? OPENCODE_CONFIG
  }

  getDefault(): AgentConfig {
    return OPENCODE_CONFIG
  }

  list(): AgentConfig[] {
    return Array.from(this.configs.values())
  }

  has(name: string): boolean {
    return this.configs.has(name)
  }
}

export const agentConfigManager = new AgentConfigManager()
