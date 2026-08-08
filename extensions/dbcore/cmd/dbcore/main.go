package main

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const version = "0.1.0"

type source struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Kind        string         `json:"kind"`
	URI         string         `json:"uri"`
	Description string         `json:"description,omitempty"`
	Status      string         `json:"status"`
	Config      map[string]any `json:"config,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
}

type store struct {
	mu   sync.Mutex
	file string
	list []source
}

func openStore(dataDir string) (*store, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	s := &store{file: filepath.Join(dataDir, "sources.json"), list: []source{}}
	raw, err := os.ReadFile(s.file)
	if err == nil {
		err = json.Unmarshal(raw, &s.list)
	}
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	return s, err
}

func (s *store) save() error {
	raw, err := json.MarshalIndent(s.list, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.file + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.file)
}

type app struct {
	token   string
	sources *store
	server  *http.Server
}

func main() {
	port := env("NEXUS_EXTENSION_PORT", "19090")
	token := strings.TrimSpace(os.Getenv("NEXUS_EXTENSION_TOKEN"))
	if token == "" {
		log.Fatal("NEXUS_EXTENSION_TOKEN is required")
	}
	sources, err := openStore(env("NEXUS_EXTENSION_DATA_DIR", ".data"))
	if err != nil {
		log.Fatal(err)
	}
	a := &app{token: token, sources: sources}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", a.index)
	mux.HandleFunc("GET /healthz", a.authorize(a.health))
	mux.HandleFunc("GET /manifest", a.authorize(a.manifest))
	mux.HandleFunc("POST /shutdown", a.authorize(a.shutdown))
	mux.HandleFunc("GET /api/v1/kinds", a.authorize(a.kinds))
	mux.HandleFunc("GET /api/v1/sources", a.authorize(a.listSources))
	mux.HandleFunc("POST /api/v1/sources", a.authorize(a.createSource))
	mux.HandleFunc("DELETE /api/v1/sources/{id}", a.authorize(a.deleteSource))
	a.server = &http.Server{Addr: "127.0.0.1:" + port, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("DbCore extension %s listening on %s", version, a.server.Addr)
	if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
func jsonResponse(w http.ResponseWriter, code int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(value)
}
func errorResponse(w http.ResponseWriter, code int, message string) {
	jsonResponse(w, code, map[string]any{"error": map[string]string{"code": http.StatusText(code), "message": message}})
}

func (a *app) authorize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(a.token)) != 1 {
			errorResponse(w, http.StatusUnauthorized, "扩展访问令牌无效")
			return
		}
		next(w, r)
	}
}

func (a *app) health(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]any{"ok": true, "extensionId": "dbcore", "version": version})
}
func (a *app) manifest(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]any{"id": "dbcore", "name": "DbCore", "version": version, "apiVersion": "1"})
}
func (a *app) shutdown(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusAccepted, map[string]bool{"ok": true})
	go func() { time.Sleep(100 * time.Millisecond); _ = a.server.Close() }()
}
func (a *app) kinds(w http.ResponseWriter, _ *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]any{"items": []map[string]string{{"id": "postgres", "label": "PostgreSQL"}, {"id": "mysql", "label": "MySQL"}, {"id": "sqlite", "label": "SQLite"}, {"id": "excel", "label": "Excel"}}})
}

func (a *app) listSources(w http.ResponseWriter, r *http.Request) {
	a.sources.mu.Lock()
	defer a.sources.mu.Unlock()
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	total := len(a.sources.list)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages < 1 {
		totalPages = 1
	}
	jsonResponse(w, http.StatusOK, map[string]any{"items": a.sources.list[start:end], "page": page, "page_size": pageSize, "total": total, "total_pages": totalPages})
}

func (a *app) createSource(w http.ResponseWriter, r *http.Request) {
	var input source
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errorResponse(w, http.StatusBadRequest, "请求体无效")
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Kind = strings.TrimSpace(input.Kind)
	if input.Name == "" || input.Kind == "" {
		errorResponse(w, http.StatusBadRequest, "名称和类型不能为空")
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	input.ID = fmt.Sprintf("src-%d", time.Now().UnixNano())
	input.Status = "active"
	input.CreatedAt = now
	input.UpdatedAt = now
	a.sources.mu.Lock()
	defer a.sources.mu.Unlock()
	a.sources.list = append(a.sources.list, input)
	if err := a.sources.save(); err != nil {
		errorResponse(w, http.StatusInternalServerError, "保存失败")
		return
	}
	jsonResponse(w, http.StatusCreated, input)
}

func (a *app) deleteSource(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	a.sources.mu.Lock()
	defer a.sources.mu.Unlock()
	for i, item := range a.sources.list {
		if item.ID == id {
			a.sources.list = append(a.sources.list[:i], a.sources.list[i+1:]...)
			if err := a.sources.save(); err != nil {
				errorResponse(w, 500, "保存失败")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	errorResponse(w, http.StatusNotFound, "数据源不存在")
}

func (a *app) index(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if subtle.ConstantTimeCompare([]byte(token), []byte(a.token)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, pageHTML, strconv.Quote(token))
}

const pageHTML = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>DbCore</title><style>
*{box-sizing:border-box}body{margin:0;font:14px -apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f7f8fa;color:#172033}.shell{display:grid;grid-template-columns:220px 1fr;min-height:100vh}.side{background:#151923;color:#fff;padding:24px}.side h1{font-size:20px;margin:0 0 28px}.side button{display:block;width:100%%;padding:11px 12px;margin:5px 0;border:0;border-radius:7px;background:transparent;color:#b9c0cc;text-align:left}.side button.active{background:#2b3242;color:#fff}.main{padding:34px;max-width:1100px}.head{display:flex;justify-content:space-between;align-items:center}.card{background:#fff;border:1px solid #e4e7ec;border-radius:10px;margin-top:22px;overflow:hidden}.empty{padding:60px;text-align:center;color:#7b8494}table{width:100%%;border-collapse:collapse}th,td{text-align:left;padding:14px;border-bottom:1px solid #edf0f3}button.primary{border:0;border-radius:7px;background:#3056d3;color:#fff;padding:10px 15px}dialog{border:0;border-radius:12px;box-shadow:0 20px 60px #0003;width:440px}label{display:block;margin:14px 0}input,select,textarea{width:100%%;padding:9px;border:1px solid #ccd2dc;border-radius:6px}</style></head><body><div class="shell"><aside class="side"><h1>DbCore</h1><button class="active">数据源</button><button disabled>SQL 查询工作台</button><button disabled>数据库建模</button><button disabled>日志采集</button></aside><main class="main"><div class="head"><div><h1>数据源</h1><p>数据保存在 DbCore 扩展自己的数据目录中。</p></div><button class="primary" onclick="editor.showModal()">新建数据源</button></div><div class="card" id="content"><div class="empty">加载中…</div></div></main></div><dialog id="editor"><form method="dialog" onsubmit="createSource(event)"><h2>新建数据源</h2><label>名称<input id="name" required></label><label>类型<select id="kind"><option value="postgres">PostgreSQL</option><option value="mysql">MySQL</option><option value="sqlite">SQLite</option><option value="excel">Excel</option></select></label><label>URI<input id="uri"></label><label>描述<textarea id="description"></textarea></label><button value="cancel">取消</button> <button class="primary" value="default">保存</button></form></dialog><script>
const token=%s, headers={'Authorization':'Bearer '+token,'Content-Type':'application/json'};async function load(){const r=await fetch('/api/v1/sources?page=1&page_size=100',{headers});const d=await r.json();content.innerHTML=d.items.length?'<table><thead><tr><th>名称</th><th>类型</th><th>URI</th><th>状态</th><th>操作</th></tr></thead><tbody>'+d.items.map(x=>'<tr><td>'+esc(x.name)+'</td><td>'+esc(x.kind)+'</td><td>'+esc(x.uri||'—')+'</td><td>'+esc(x.status)+'</td><td><button onclick="removeSource(\''+x.id+'\')">删除</button></td></tr>').join('')+'</tbody></table>':'<div class="empty">暂无数据源</div>'}function esc(v){return String(v).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]))}async function createSource(e){e.preventDefault();await fetch('/api/v1/sources',{method:'POST',headers,body:JSON.stringify({name:name.value,kind:kind.value,uri:uri.value,description:description.value})});editor.close();e.target.reset();load()}async function removeSource(id){if(confirm('确定删除？')){await fetch('/api/v1/sources/'+encodeURIComponent(id),{method:'DELETE',headers});load()}}load();</script></body></html>`
