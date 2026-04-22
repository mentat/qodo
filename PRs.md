# Qodo.ai PR Review Test Corpus

Ten branches, each off `main`, each introducing **one obvious bug**. Used to measure qodo.ai's recall across categories. All bugs are deliberate. Commit messages intentionally describe the change neutrally (as a "fix", "simplify", etc.) so qodo.ai isn't tipped off.

Open a PR from each branch against `main` and grade qodo.ai on whether it flags the expected finding.

| # | Branch | File(s) | Category | Expected finding |
|---|---|---|---|---|
| 1 | `test/bug-01-auth-bypass-patch` | `api/services/todos.go` | Security — broken access control | `TodoService.Patch` no longer verifies the target todo belongs to `userID`. Any authenticated user can PATCH any todo ID. |
| 2 | `test/bug-02-cross-tenant-list` | `api/services/todos.go` | Security — tenant isolation | `List` drops the `userId` filter when a priority filter is set, leaking every user's todos of that priority. |
| 3 | `test/bug-03-xss-description` | `frontend/src/components/TodoItem.tsx` | Security — XSS | Todo description is rendered via `dangerouslySetInnerHTML`, allowing stored XSS. |
| 4 | `test/bug-04-no-title-validation` | `api/services/todos.go` | Input validation | `Create` no longer rejects empty/whitespace titles; invariant violated. |
| 5 | `test/bug-05-stale-useeffect-deps` | `frontend/src/components/TodoForm.tsx` | React correctness | `useEffect` deps changed from `[todo]` to `[]`; editing a different todo shows the previous todo's values. |
| 6 | `test/bug-06-token-never-refreshed` | `frontend/src/api/todos.ts` | Auth — token staleness | Firebase ID token is cached in a module-level variable forever; expired tokens are never refreshed. |
| 7 | `test/bug-07-position-race` | `api/services/todos.go` | Concurrency — race condition | Read-then-write "next position" with an added 100ms `time.Sleep`; concurrent creates collide on the same position. |
| 8 | `test/bug-08-completed-case-sensitive` | `api/handlers/todo.go` | Logic bug | `completed` query param now returns `true` for any non-empty value (including `completed=false`). |
| 9 | `test/bug-09-swallowed-batch-error` | `api/chat/store.go` | Error handling | `Clear` swallows batch-commit errors mid-loop via `_ = err`; partial deletions succeed silently. |
| 10 | `test/bug-10-breaking-field-rename` | `api/services/todos.go` | API contract / breaking change | `DueDate` JSON tag renamed from `dueDate` to `due_date`; frontend (`frontend/src/types/todo.ts`) still reads `dueDate`. |

## How to run the test

```sh
# For each branch:
git push -u origin test/bug-NN-<slug>
gh pr create --base main --head test/bug-NN-<slug> \
  --title "<neutral title>" --body "<neutral body>"

# Wait for qodo.ai's review, then score:
#   - hit:  qodo.ai flagged the expected finding
#   - miss: qodo.ai said LGTM or flagged something unrelated
#   - partial: qodo.ai flagged the right file/area but not the specific defect
```

## Scoring template

| # | Branch | Result | Notes |
|---|---|---|---|
| 1 | auth-bypass-patch | | |
| 2 | cross-tenant-list | | |
| 3 | xss-description | | |
| 4 | no-title-validation | | |
| 5 | stale-useeffect-deps | | |
| 6 | token-never-refreshed | | |
| 7 | position-race | | |
| 8 | completed-case-sensitive | | |
| 9 | swallowed-batch-error | | |
| 10 | breaking-field-rename | | |

Recall = hits / 10.
