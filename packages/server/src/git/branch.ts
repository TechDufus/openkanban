import { git, exec } from "./utils"

export async function getDefaultBranch(repoPath: string): Promise<string> {
  try {
    const ref = await git(["symbolic-ref", "refs/remotes/origin/HEAD"], repoPath)
    return ref.replace("refs/remotes/origin/", "")
  } catch {
    for (const branch of ["main", "master"]) {
      const result = await exec(["git", "rev-parse", "--verify", branch], repoPath)
      if (result.exitCode === 0) {
        return branch
      }
    }
    return "main"
  }
}

export async function branchExists(repoPath: string, branch: string): Promise<boolean> {
  const result = await exec(["git", "rev-parse", "--verify", branch], repoPath)
  return result.exitCode === 0
}

export async function hasUncommittedChanges(worktreePath: string): Promise<boolean> {
  const result = await exec(["git", "status", "--porcelain"], worktreePath)
  if (result.exitCode !== 0) {
    return false
  }
  return result.stdout.trim().length > 0
}

export async function deleteBranch(repoPath: string, branchName: string, force = false): Promise<void> {
  const flag = force ? "-D" : "-d"
  await git(["branch", flag, branchName], repoPath)
}

export async function getCurrentBranch(repoPath: string): Promise<string | null> {
  try {
    return await git(["rev-parse", "--abbrev-ref", "HEAD"], repoPath)
  } catch {
    return null
  }
}
