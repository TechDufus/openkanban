import fs from "node:fs"
import path from "node:path"
import { git, exec, branchToWorktreeName } from "./utils"

export interface WorktreeInfo {
  path: string
  head: string
  branch: string | null
}

export async function createWorktree(
  repoPath: string,
  branchName: string,
  baseBranch: string,
  worktreeBase: string
): Promise<string> {
  const worktreeDirName = branchToWorktreeName(branchName)
  const worktreePath = path.join(worktreeBase, worktreeDirName)

  fs.mkdirSync(worktreeBase, { recursive: true })

  if (fs.existsSync(worktreePath)) {
    return worktreePath
  }

  await git(["worktree", "add", "-b", branchName, worktreePath, baseBranch], repoPath)

  return worktreePath
}

export async function removeWorktree(
  repoPath: string,
  worktreePath: string,
  force = false
): Promise<void> {
  const args = ["worktree", "remove", worktreePath]
  if (force) {
    args.push("--force")
  }

  const result = await exec(["git", ...args], repoPath)

  if (result.exitCode !== 0 && !result.stderr.includes("not a working tree")) {
    throw new Error(`Failed to remove worktree: ${result.stderr}`)
  }

  if (fs.existsSync(worktreePath)) {
    fs.rmSync(worktreePath, { recursive: true, force: true })
  }
}

export async function listWorktrees(repoPath: string): Promise<WorktreeInfo[]> {
  const output = await git(["worktree", "list", "--porcelain"], repoPath)
  return parseWorktreeList(output)
}

function parseWorktreeList(output: string): WorktreeInfo[] {
  const worktrees: WorktreeInfo[] = []
  let current: Partial<WorktreeInfo> = {}

  for (const line of output.split("\n")) {
    if (line.startsWith("worktree ")) {
      if (current.path) {
        worktrees.push({
          path: current.path,
          head: current.head ?? "",
          branch: current.branch ?? null,
        })
      }
      current = { path: line.slice(9) }
    } else if (line.startsWith("HEAD ")) {
      current.head = line.slice(5)
    } else if (line.startsWith("branch ")) {
      current.branch = line.slice(7).replace("refs/heads/", "")
    }
  }

  if (current.path) {
    worktrees.push({
      path: current.path,
      head: current.head ?? "",
      branch: current.branch ?? null,
    })
  }

  return worktrees
}

export async function worktreeExists(repoPath: string, worktreePath: string): Promise<boolean> {
  const worktrees = await listWorktrees(repoPath)
  return worktrees.some((wt) => wt.path === worktreePath)
}

export async function pruneWorktrees(repoPath: string): Promise<void> {
  await git(["worktree", "prune"], repoPath)
}
