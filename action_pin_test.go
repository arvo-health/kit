package kit_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// placeholderJobToken stands in for the job's GITHUB_TOKEN. It is not a credential.
const placeholderJobToken = "placeholder-job-token"

const attackerTag = "v99.0.0-compromised"

var fullCommitSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)

// actionStep is one uses: line in a job whose GITHUB_TOKEN can create git refs.
type actionStep struct {
	file  string
	job   string
	uses  string
	name  string
	ref   string
	with  map[string]string
	perms map[string]string
}

// TestRetargetedTagInContentsWriteJobPublishesRelease fails while a push-to-main
// job with contents: write checks out an action by a movable ref. Retargeting that
// ref runs the new tree with the job token and publishes a module tag. Pinning
// every such uses: value to a full commit SHA makes the same sequence publish nothing.
func TestRetargetedTagInContentsWriteJobPublishesRelease(t *testing.T) {
	steps := contentsWritePushSteps(t)
	if len(steps) == 0 {
		t.Fatal("expected .github/workflows/tag.yml to run a contents: write job on push to main")
	}
	for _, step := range steps {
		step := step
		t.Run(step.uses, func(t *testing.T) {
			published := retargetAndRun(t, step, fullCommitSHA.MatchString(step.ref))
			if fullCommitSHA.MatchString(step.ref) {
				if !upstreamObjectExists(t, step.name, step.ref) {
					t.Fatalf("uses %s pins %s, which is not an object in that repository", step.uses, step.ref)
				}
				if published != "" {
					t.Fatalf("full SHA %s still published %s after a tag moved", step.uses, published)
				}
				return
			}
			if published == "" {
				t.Fatalf("movable ref %s did not publish; the reproduction did not exercise the job token", step.uses)
			}
			t.Fatalf("retargeting %s published %s using the contents: write job token; pin this action to a 40-character commit SHA", step.uses, published)
		})
	}
}

// TestCommitSHARefIgnoresRetargetedTag is the same runner sequence as the workflow
// test, with the ref fixed to a commit object. Moving a tag does not change the tree
// that runs, so nothing is published.
func TestCommitSHARefIgnoresRetargetedTag(t *testing.T) {
	step := actionStep{
		uses:  "example/action@sha",
		name:  "example/action",
		ref:   "", // filled with the benign commit SHA inside the harness
		with:  map[string]string{},
		perms: map[string]string{"contents": "write"},
	}
	if published := retargetAndRun(t, step, true); published != "" {
		t.Fatalf("commit SHA ref published %s after the tag moved", published)
	}
}

// TestContentsReadDoesNotCreateReleaseObjects refuses a create-blob call before
// any object is written when the job token's contents permission is not write.
func TestContentsReadDoesNotCreateReleaseObjects(t *testing.T) {
	repo := initRepo(t)
	before := git(t, repo, "count-objects", "-v")
	api := newContentsServer(t, repo, map[string]string{"contents": "read"})
	defer api.Close()

	status, body := postJSON(t, api.URL+"/repos/arvo-health/kit/git/blobs", map[string]string{
		"content":  "compromised\n",
		"encoding": "utf-8",
	})
	if status != http.StatusForbidden {
		t.Fatalf("contents: read status = %d body %s", status, body)
	}
	if after := git(t, repo, "count-objects", "-v"); after != before {
		t.Fatalf("contents: read wrote git objects\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if tags := git(t, repo, "tag", "--list"); tags != "" {
		t.Fatalf("contents: read created tags %q", tags)
	}
}

func contentsWritePushSteps(t *testing.T) []actionStep {
	t.Helper()
	var files []string
	for _, pattern := range []string{".github/workflows/*.yml", ".github/workflows/*.yaml"} {
		found, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, found...)
	}
	if len(files) == 0 {
		t.Fatal("no workflow files")
	}
	var steps []actionStep
	for _, file := range files {
		steps = append(steps, stepsIn(t, file)...)
	}
	return steps
}

func stepsIn(t *testing.T, file string) []actionStep {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	if !pushIncludesMain(doc["on"]) {
		return nil
	}
	workflowPerms, workflowSet := permissionMap(doc["permissions"])
	jobs, _ := doc["jobs"].(map[string]any)
	var steps []actionStep
	for jobName, jobRaw := range jobs {
		job, _ := jobRaw.(map[string]any)
		if job == nil {
			continue
		}
		perms, set := permissionMap(job["permissions"])
		if !set {
			perms, set = workflowPerms, workflowSet
		}
		if !set || perms["contents"] != "write" {
			continue
		}
		for _, stepRaw := range asSlice(job["steps"]) {
			step, _ := stepRaw.(map[string]any)
			if step == nil {
				continue
			}
			uses, _ := step["uses"].(string)
			name, ref, ok := parseUses(uses)
			if !ok {
				continue
			}
			with := map[string]string{}
			if inputs, ok := step["with"].(map[string]any); ok {
				for key, value := range inputs {
					with[fmt.Sprint(key)] = fmt.Sprint(value)
				}
			}
			steps = append(steps, actionStep{
				file:  file,
				job:   jobName,
				uses:  uses,
				name:  name,
				ref:   ref,
				with:  with,
				perms: perms,
			})
		}
	}
	return steps
}

func pushIncludesMain(on any) bool {
	switch value := on.(type) {
	case string:
		return value == "push"
	case []any:
		for _, item := range value {
			if fmt.Sprint(item) == "push" {
				return true
			}
		}
		return false
	case map[string]any:
		push, ok := value["push"]
		if !ok {
			return false
		}
		if push == nil {
			return true
		}
		spec, ok := push.(map[string]any)
		if !ok {
			return true
		}
		branches, ok := spec["branches"]
		if !ok {
			return true
		}
		for _, branch := range asSlice(branches) {
			name := fmt.Sprint(branch)
			if name == "main" || name == "*" || name == "**" {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func permissionMap(raw any) (map[string]string, bool) {
	switch value := raw.(type) {
	case nil:
		return nil, false
	case string:
		switch value {
		case "write-all":
			return map[string]string{"contents": "write"}, true
		case "read-all":
			return map[string]string{"contents": "read"}, true
		default:
			return map[string]string{}, true
		}
	case map[string]any:
		out := map[string]string{}
		for key, item := range value {
			out[fmt.Sprint(key)] = fmt.Sprint(item)
		}
		return out, true
	default:
		return nil, false
	}
}

func asSlice(raw any) []any {
	switch value := raw.(type) {
	case []any:
		return value
	case nil:
		return nil
	default:
		return []any{value}
	}
}

func parseUses(uses string) (name, ref string, ok bool) {
	uses = strings.TrimSpace(uses)
	if uses == "" || strings.HasPrefix(uses, "./") || strings.HasPrefix(uses, "docker://") {
		return "", "", false
	}
	at := strings.LastIndex(uses, "@")
	if at <= 0 {
		return uses, "", true
	}
	return uses[:at], uses[at+1:], true
}

// retargetAndRun builds an action repository, points a tag at a benign tree,
// runs it, moves the tag to a tree that calls the contents API, and runs the
// ref from the workflow again. A full commit SHA keeps resolving to the first tree.
func retargetAndRun(t *testing.T, step actionStep, shaPin bool) string {
	t.Helper()
	actionRepo := initRepo(t)
	benign := commitAction(t, actionRepo, benignScript)
	malicious := commitAction(t, actionRepo, maliciousScript)
	ref := step.ref
	if shaPin {
		// A full SHA is an object id. Resolve that kind of ref locally as the benign
		// commit's id; the workflow test separately checks the pinned id exists upstream.
		ref = benign
	} else {
		git(t, actionRepo, "tag", ref, benign)
	}

	target := initRepo(t)
	api := newContentsServer(t, target, step.perms)
	defer api.Close()

	runResolved := func() {
		t.Helper()
		resolved := resolveRef(t, actionRepo, ref)
		if shaPin && resolved != benign {
			t.Fatalf("SHA ref %s resolved to %s", ref, resolved)
		}
		dir := t.TempDir()
		extract(t, actionRepo, resolved, dir)
		runAction(t, dir, step, target, api.URL)
	}

	runResolved()
	if tags := git(t, target, "tag", "--list"); tags != "" {
		t.Fatalf("first resolution published %q before the tag moved", tags)
	}
	git(t, actionRepo, "tag", "-f", "moved", malicious)
	if !shaPin {
		git(t, actionRepo, "tag", "-f", step.ref, malicious)
	}
	runResolved()
	listed := git(t, target, "tag", "--list", attackerTag)
	if listed == "" {
		return ""
	}
	body := git(t, target, "show", attackerTag+":compromised.txt")
	if body != "compromised" {
		t.Fatalf("tag %s does not point at the attacker commit (%q)", attackerTag, body)
	}
	return listed
}

func resolveRef(t *testing.T, repo, ref string) string {
	t.Helper()
	if fullCommitSHA.MatchString(ref) {
		return git(t, repo, "rev-parse", "--verify", ref+"^{commit}")
	}
	if sha, err := gitOutput(repo, "rev-parse", "--verify", "refs/tags/"+ref+"^{commit}"); err == nil {
		return sha
	}
	return git(t, repo, "rev-parse", "--verify", "refs/heads/"+ref+"^{commit}")
}

func commitAction(t *testing.T, repo, script string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, "action.yml"), []byte(attackerManifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "main.js"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, repo, "add", "action.yml", "main.js")
	git(t, repo, "commit", "-m", "action")
	return git(t, repo, "rev-parse", "HEAD")
}

func runAction(t *testing.T, dir string, step actionStep, target, apiURL string) {
	t.Helper()
	manifestRaw, err := os.ReadFile(filepath.Join(dir, "action.yml"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Inputs map[string]struct {
			Default string `yaml:"default"`
		} `yaml:"inputs"`
		Runs struct {
			Using string `yaml:"using"`
			Main  string `yaml:"main"`
		} `yaml:"runs"`
	}
	if err := yaml.Unmarshal(manifestRaw, &manifest); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(manifest.Runs.Using, "node") || manifest.Runs.Main == "" {
		t.Fatalf("action %s runs.using=%q main=%q", step.uses, manifest.Runs.Using, manifest.Runs.Main)
	}
	inputs := map[string]string{}
	for key, value := range step.with {
		inputs[key] = evaluateJobExpr(value)
	}
	for key, input := range manifest.Inputs {
		if _, set := inputs[key]; set {
			continue
		}
		inputs[key] = evaluateJobExpr(input.Default)
	}
	head := git(t, target, "rev-parse", "HEAD")
	cmd := exec.Command("node", filepath.Join(dir, manifest.Runs.Main))
	cmd.Dir = dir
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"GITHUB_REPOSITORY=arvo-health/kit",
		"GITHUB_SHA=" + head,
		"CONTENTS_API_BASE=" + apiURL,
	}
	for key, value := range inputs {
		env = append(env, "INPUT_"+strings.ToUpper(strings.ReplaceAll(key, " ", "_"))+"="+value)
	}
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node %s: %v\n%s", manifest.Runs.Main, err, out)
	}
}

func evaluateJobExpr(value string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "${{ github.token }}", "${{ secrets.GITHUB_TOKEN }}":
		return placeholderJobToken
	default:
		return value
	}
}

func upstreamObjectExists(t *testing.T, nameWithOwner, sha string) bool {
	t.Helper()
	for _, kind := range []string{"commits", "tags"} {
		url := fmt.Sprintf("https://api.github.com/repos/%s/git/%s/%s", nameWithOwner, kind, sha)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("User-Agent", "kit-review")
		req.Header.Set("Accept", "application/vnd.github+json")
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode == http.StatusOK {
			return true
		}
	}
	return false
}

func newContentsServer(t *testing.T, repo string, perms map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+placeholderJobToken {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		// Contents write is required for blobs, trees, commits, and refs.
		// Refuse before any git write.
		if perms["contents"] != "write" {
			http.Error(w, "Resource not accessible by integration", http.StatusForbidden)
			return
		}
		var payload map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&payload)
		}
		fail := func(err error) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/blobs"):
			content, _ := payload["content"].(string)
			sha, err := gitStdinOutput(repo, content, "hash-object", "-w", "--stdin")
			if err != nil {
				fail(err)
				return
			}
			writeJSON(w, map[string]string{"sha": sha})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/trees"):
			entries, _ := payload["tree"].([]any)
			var spec strings.Builder
			for _, entry := range entries {
				item, _ := entry.(map[string]any)
				fmt.Fprintf(&spec, "%s %s %s\t%s\n", item["mode"], item["type"], item["sha"], item["path"])
			}
			sha, err := gitStdinOutput(repo, spec.String(), "mktree")
			if err != nil {
				fail(err)
				return
			}
			writeJSON(w, map[string]string{"sha": sha})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/commits"):
			tree, _ := payload["tree"].(string)
			message, _ := payload["message"].(string)
			args := []string{"commit-tree", tree, "-m", message}
			for _, parent := range asSlice(payload["parents"]) {
				args = append(args, "-p", fmt.Sprint(parent))
			}
			sha, err := gitOutput(repo, args...)
			if err != nil {
				fail(err)
				return
			}
			writeJSON(w, map[string]string{"sha": sha})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/git/refs"):
			ref, _ := payload["ref"].(string)
			sha, _ := payload["sha"].(string)
			if !strings.HasPrefix(ref, "refs/tags/") {
				http.Error(w, "ref is not a tag", http.StatusUnprocessableEntity)
				return
			}
			if _, err := gitOutput(repo, "update-ref", ref, sha); err != nil {
				fail(err)
				return
			}
			writeJSON(w, map[string]string{"ref": ref, "sha": sha})
		default:
			http.NotFound(w, r)
		}
	}))
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func postJSON(t *testing.T, url string, body any) (int, string) {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+placeholderJobToken)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(data)
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	git(t, dir, "config", "user.email", "test@example.com")
	git(t, dir, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("module\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "README")
	git(t, dir, "commit", "-m", "init")
	return dir
}

func extract(t *testing.T, repo, sha, dest string) {
	t.Helper()
	archive, err := exec.Command("git", "-C", repo, "archive", sha).Output()
	if err != nil {
		t.Fatal(err)
	}
	tar := exec.Command("tar", "-x", "-C", dest)
	tar.Stdin = bytes.NewReader(archive)
	if out, err := tar.CombinedOutput(); err != nil {
		t.Fatalf("tar: %v\n%s", err, out)
	}
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitOutput(dir, args...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func gitStdinOutput(dir, stdin string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitEnv()
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitEnv()
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func gitEnv() []string {
	return append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
}

const attackerManifest = `name: replacement
inputs:
  github_token:
    default: ${{ github.token }}
  token:
    default: ${{ github.token }}
runs:
  using: node20
  main: main.js
`

const benignScript = `process.exit(0);
`

const maliciousScript = `
const http = require('http');
const https = require('https');
const {URL} = require('url');

function request(method, pathname, body) {
  const base = new URL(process.env.CONTENTS_API_BASE);
  const lib = base.protocol === 'https:' ? https : http;
  const data = JSON.stringify(body);
  return new Promise((resolve, reject) => {
    const req = lib.request({
      protocol: base.protocol,
      hostname: base.hostname,
      port: base.port,
      method,
      path: pathname,
      headers: {
        Authorization: 'Bearer ' + (process.env.INPUT_GITHUB_TOKEN || process.env.INPUT_TOKEN || ''),
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(data),
      },
    }, (res) => {
      const chunks = [];
      res.on('data', (chunk) => chunks.push(chunk));
      res.on('end', () => {
        const text = Buffer.concat(chunks).toString('utf8');
        if (res.statusCode >= 300) {
          reject(new Error(res.statusCode + ' ' + text));
          return;
        }
        resolve(text ? JSON.parse(text) : {});
      });
    });
    req.on('error', reject);
    req.write(data);
    req.end();
  });
}

(async () => {
  const repo = '/repos/' + process.env.GITHUB_REPOSITORY;
  const blob = await request('POST', repo + '/git/blobs', {content: 'compromised\n', encoding: 'utf-8'});
  const tree = await request('POST', repo + '/git/trees', {
    tree: [{path: 'compromised.txt', mode: '100644', type: 'blob', sha: blob.sha}],
  });
  const commit = await request('POST', repo + '/git/commits', {
    message: 'compromised release',
    tree: tree.sha,
    parents: [process.env.GITHUB_SHA],
  });
  await request('POST', repo + '/git/refs', {ref: 'refs/tags/` + attackerTag + `', sha: commit.sha});
})().catch((err) => {
  console.error(err);
  process.exit(1);
});
`
