package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTheRouteAnswers is the permit half: the path openapi.json declares
// answers 200 and the line the settings hold.
func TestTheRouteAnswers(t *testing.T) {
	recorder := httptest.NewRecorder()
	routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/autostorager", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /autostorager answered %d, want %d", recorder.Code, http.StatusOK)
	}

	var body struct {
		Message string `json:"message"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("the answer is not JSON: %v", err)
	}

	if body.Message != defaultMessage {
		t.Fatalf("the answer reads %q, want %q", body.Message, defaultMessage)
	}
}

// TestAPathThisModuleDoesNotServeIsRefusedByCode is the refusal half. A client
// branches on the code, so the test reads the code and not the sentence: a
// reworded sentence must not break a caller, and a changed code must.
func TestAPathThisModuleDoesNotServeIsRefusedByCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/nothing-here", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("an unserved path answered %d, want %d", recorder.Code, http.StatusNotFound)
	}

	if got := recorder.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("the refusal is %s, want application/problem+json", got)
	}

	var problem struct {
		Code   string            `json:"code"`
		Title  string            `json:"title"`
		Detail string            `json:"detail"`
		Params map[string]string `json:"params"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &problem); err != nil {
		t.Fatalf("the refusal is not JSON: %v", err)
	}

	if want := RouteNotFound.Code().String(); problem.Code != want {
		t.Fatalf("the refusal carries %q, want %q", problem.Code, want)
	}

	// The English is kept beside the code, so a log line and an untranslated
	// screen still read.
	if problem.Title == "" || problem.Detail == "" {
		t.Fatalf("the refusal lost its English: title %q, detail %q", problem.Title, problem.Detail)
	}

	if problem.Params["path"] != "/nothing-here" {
		t.Fatalf("the refusal names the path %q, want /nothing-here", problem.Params["path"])
	}
}

// TestTheManifestPublishesTheCodesDefault holds module.yaml and this code to
// one default. An operator reads the manifest to learn what a setting does when
// they write nothing, so a manifest that publishes a different number than the
// code uses is documentation that lies.
func TestTheManifestPublishesTheCodesDefault(t *testing.T) {
	manifest, err := os.ReadFile("module.yaml")
	if err != nil {
		t.Fatalf("read module.yaml: %v", err)
	}

	declared := `default: "` + defaultMessage + `"`
	if !strings.Contains(string(manifest), declared) {
		t.Fatalf("module.yaml does not declare %s for the message setting", declared)
	}
}

// TestTheLanguageListMatchesTheFiles keeps the offered languages and the files
// that back them in step. A language added to ui/src/locales or ui/src/manual
// without a line here would be unreadable from the settings page, and a tag
// listed without files would be a promise the build cannot keep.
func TestTheLanguageListMatchesTheFiles(t *testing.T) {
	t.Parallel()

	for _, dir := range []string{"ui/src/locales", "ui/src/manual"} {
		files, err := filepath.Glob(filepath.Join(dir, "*.json"))
		if err != nil {
			t.Fatalf("list the files under %s: %v", dir, err)
		}

		shipped := make(map[string]bool, len(files))

		for _, file := range files {
			shipped[strings.TrimSuffix(filepath.Base(file), ".json")] = true
		}

		for _, tag := range languages {
			if tag == defaultLanguage {
				continue
			}

			if !shipped[tag] {
				t.Errorf("the settings offer %q, but %s/%s.json is not there", tag, dir, tag)
			}

			delete(shipped, tag)
		}

		for tag := range shipped {
			t.Errorf("%s/%s.json exists, but the settings do not offer it", dir, tag)
		}
	}
}

// TestNoSettingIsReadFromTheEnvironment keeps every setting in the manifest. A
// setting read from the environment is one an operator cannot find in the
// manifest, cannot see on the settings pane, and cannot set through the API.
func TestNoSettingIsReadFromTheEnvironment(t *testing.T) {
	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}

	if strings.Contains(string(source), "os.Getenv") {
		t.Fatal("main.go reads an environment variable. Declare the setting in module.yaml instead")
	}
}
