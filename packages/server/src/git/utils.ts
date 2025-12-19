import { spawn } from "bun"

export interface ExecResult {
  stdout: string
  stderr: string
  exitCode: number
}

export async function exec(
  command: string[],
  cwd?: string
): Promise<ExecResult> {
  const proc = spawn(command, {
    cwd,
    stdout: "pipe",
    stderr: "pipe",
  })

  const stdout = await new Response(proc.stdout).text()
  const stderr = await new Response(proc.stderr).text()
  const exitCode = await proc.exited

  return { stdout, stderr, exitCode }
}

export async function git(args: string[], cwd?: string): Promise<string> {
  const result = await exec(["git", ...args], cwd)
  if (result.exitCode !== 0) {
    throw new Error(`git ${args[0]} failed: ${result.stderr || result.stdout}`)
  }
  return result.stdout.trim()
}

export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 50)
}

export function sanitizeBranchName(name: string): string {
  let sanitized = name
    .replace(/^refs\/heads\//, "")
    .replace(/^(agent|feature|task)\//, "")
    .replace(/\//g, "-")
    .replace(/[^a-zA-Z0-9-_]/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")

  return sanitized || "unnamed"
}

export function branchToWorktreeName(branchName: string): string {
  return sanitizeBranchName(branchName)
}
