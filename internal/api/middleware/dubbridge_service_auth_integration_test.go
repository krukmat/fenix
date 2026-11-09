package middleware_test

import (
    "context"
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "path/filepath"
    "strings"
    "testing"
    "time"

    "github.com/matiasleandrokruk/fenix/internal/api/middleware"
    domainauth "github.com/matiasleandrokruk/fenix/internal/domain/auth"
    fxsqlite "github.com/matiasleandrokruk/fenix/internal/infra/sqlite"
    pkgauth "github.com/matiasleandrokruk/fenix/pkg/auth"
)

// B2.A3.T3: true Fenix modernc SQLite and credential store; inert HTTP handler only.

func t3RealStoreDB(t *testing.T) (*sql.DB, *domainauth.ServiceCredentialStore) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "service-identity.db")
	db, err := fxsqlite.NewDB(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := fxsqlite.MigrateUp(db); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO workspace(id,name,slug,created_at,updated_at)
        VALUES ('w1','One','one','2026-01-01','2026-01-01'),
               ('w2','Two','two','2026-01-01','2026-01-01')`)
	if err != nil {
		t.Fatal(err)
	}
	return db, domainauth.NewServiceCredentialStore(db)
}



func t3Identity(t *testing.T, store *domainauth.ServiceCredentialStore, id, ws, org, project string) {
    t.Helper()
    identity:=domainauth.ServiceIdentity{
        ServicePrincipal:domainauth.ServicePrincipal{ID:id,Consumer:"dubbridge",WorkspaceID:ws},
        DubBridgeOrgID:org,DubBridgeProjectID:project}
    if err:=store.RegisterServicePrincipal(context.Background(),identity,time.Now().UTC().Add(-3*time.Hour));err!=nil {t.Fatal(err)}
}
func t3Credential(t *testing.T,store *domainauth.ServiceCredentialStore,id,principal string,issued,expires time.Time)string{
    t.Helper()
    token,digest,err:=domainauth.GenerateServiceToken()
    if err!=nil {t.Fatal(err)}
    if err=store.StoreServiceCredential(context.Background(),id,principal,digest,issued,expires);err!=nil {t.Fatal(err)}
    return token
}
func t3JSON(org,project,padding string)string{
    source:=map[string]string{"product":"dubbridge","org_id":org}
    if project!=""{source["project_id"]=project}
    payload:=map[string]any{
       "contract_version":"1","operation_id":"operation_123456789",
       "capability_key":"integration.diagnostic","input_schema_version":"1",
       "trace_id":"trace_12345678","source":source,
       "input":map[string]string{"padding":padding}}
    raw,_:=json.Marshal(payload)
    return string(raw)
}
func t3Req(method,token,body string)*http.Request{
    path:="/integrations/dubbridge/operations"
    if method=="GET"{path+="/operation_123456789"}
    r:=httptest.NewRequest(method,path,strings.NewReader(body))
    r.Header.Set("Authorization","Bearer "+token)
    if method=="POST"{r.Header.Set("Content-Type","application/json")}
    return r
}
type t3Downstream struct{calls int; scope middleware.DubBridgeOperationScope; authorized bool}
func (s *t3Downstream)ServeHTTP(w http.ResponseWriter,r *http.Request){
    s.calls++
    ident,authenticated:=middleware.VerifiedDubBridgeIdentity(r.Context())
    scope,authorized:=middleware.AuthorizedDubBridgeScope(r.Context())
    s.scope=scope
    s.authorized=authenticated&&authorized&&ident.ID==scope.PrincipalID&&
        ident.WorkspaceID==scope.WorkspaceID&&ident.DubBridgeOrgID==scope.OrgID&&
        ident.DubBridgeProjectID==scope.ProjectID
    w.WriteHeader(http.StatusAccepted)
}
func t3Check(t *testing.T,db *sql.DB,store *domainauth.ServiceCredentialStore,
    r *http.Request,expectedCode,expectedCalls int,bearer string)middleware.DubBridgeOperationScope{
    t.Helper()
    var before,after int
    if err:=db.QueryRow("SELECT COUNT(*) FROM agent_run").Scan(&before);err!=nil{t.Fatal(err)}
    spy:=&t3Downstream{}
    w:=httptest.NewRecorder()
    middleware.DubBridgeServiceAuth(store,nil,spy).ServeHTTP(w,r)
    if w.Code!=expectedCode||spy.calls!=expectedCalls{t.Fatalf("status=%d calls=%d expected=%d/%d",w.Code,spy.calls,expectedCode,expectedCalls)}
    if err:=db.QueryRow("SELECT COUNT(*) FROM agent_run").Scan(&after);err!=nil{t.Fatal(err)}
    if before!=after{t.Fatal("unexpected agent_run write")}
    if expectedCalls==1&&!spy.authorized{t.Fatal("unverified service identity reached handler")}
    if expectedCalls==0{
        var response map[string]any
        if err:=json.Unmarshal(w.Body.Bytes(),&response);err!=nil||response["error_code"]==nil{t.Fatalf("unsafe denial: %v",err)}
        if strings.Contains(w.Body.String(),bearer)||strings.Contains(w.Body.String(),"org-one"){t.Fatal("identity/secret disclosed")}
    }
    return spy.scope
}
func TestB2A3T3SQLiteHTTPIsolation(t *testing.T){
    db,store:=t3RealStoreDB(t)
    t3Identity(t,store,"p1","w1","org-one","project-one")
    t3Identity(t,store,"p2","w2","org-two","project-two")
    now:=time.Now().UTC()
    a:=t3Credential(t,store,"c1","p1",now.Add(-time.Hour),now.Add(time.Hour))
    b:=t3Credential(t,store,"c2","p2",now.Add(-time.Hour),now.Add(time.Hour))
    r:=t3Req("POST",a,t3JSON("org-one","project-one",""))
    r.Header.Set("X-Workspace-ID","w2")
    r.URL.RawQuery="workspace_id=w2"
    scope:=t3Check(t,db,store,r,202,1,a)
    if scope.PrincipalID!="p1"||scope.WorkspaceID!="w1"||scope.OrgID!="org-one"||scope.ProjectID!="project-one"{t.Fatal("caller scope promoted")}
    for _,x:=range []struct{token,org,project string}{
       {a,"org-two","project-one"},{a,"org-one","project-two"},
       {a,"org-one",""},{b,"org-one","project-one"},
    }{t3Check(t,db,store,t3Req("POST",x.token,t3JSON(x.org,x.project,"")),403,0,x.token)}
    // Without the real B2.C operation resolver, GET/status remains default-deny.
    t3Check(t,db,store,t3Req("GET",a,""),403,0,a)
}
func TestB2A3T3SQLiteOrgOnlyAndCredentialLifecycle(t *testing.T){
    db,store:=t3RealStoreDB(t)
    t3Identity(t,store,"p1","w1","org-one","")
    now:=time.Now().UTC()
    old:=t3Credential(t,store,"c1","p1",now.Add(-time.Hour),now.Add(time.Hour))
    body:=t3JSON("org-one","","")
    t3Check(t,db,store,t3Req("POST",old,body),202,1,old)
    t3Check(t,db,store,t3Req("POST",old,t3JSON("org-one","project-one","")),403,0,old)
    fresh,digest,err:=domainauth.GenerateServiceToken()
    if err!=nil{t.Fatal(err)}
    if err=store.RotateServiceCredential(context.Background(),"c2","p1",digest,now.Add(-time.Minute),now.Add(time.Hour));err!=nil{t.Fatal(err)}
    t3Check(t,db,store,t3Req("POST",old,body),401,0,old)
    t3Check(t,db,store,t3Req("POST",fresh,body),202,1,fresh)
    _,duplicate,err:=domainauth.GenerateServiceToken()
    if err!=nil{t.Fatal(err)}
    if err=store.RotateServiceCredential(context.Background(),"c1","p1",duplicate,now,now.Add(time.Hour));err==nil{t.Fatal("failed rotation should roll back")}
    t3Check(t,db,store,t3Req("POST",fresh,body),202,1,fresh)
    if err=store.RevokeServiceCredential(context.Background(),"c2",now);err!=nil{t.Fatal(err)}
    t3Check(t,db,store,t3Req("POST",fresh,body),401,0,fresh)
}
func TestB2A3T3SQLiteNegativeTokensAndWire(t *testing.T){
    for _,mode:=range []string{"expired","future","principalRevoked","unknown"}{
        t.Run(mode,func(t *testing.T){
            db,store:=t3RealStoreDB(t)
            t3Identity(t,store,"p1","w1","org-one","project-one")
            now:=time.Now().UTC()
            issue,expiry:=now.Add(-time.Hour),now.Add(time.Hour)
            if mode=="expired"{issue,expiry=now.Add(-2*time.Hour),now.Add(-time.Hour)}
            if mode=="future"{issue,expiry=now.Add(time.Hour),now.Add(2*time.Hour)}
            token:=t3Credential(t,store,"c1","p1",issue,expiry)
            if mode=="principalRevoked"{
                if err:=store.RevokeServicePrincipal(context.Background(),"p1",now);err!=nil{t.Fatal(err)}
            }
            if mode=="unknown"{
                var err error
                token,_,err=domainauth.GenerateServiceToken()
                if err!=nil{t.Fatal(err)}
            }
            t3Check(t,db,store,t3Req("POST",token,t3JSON("org-one","project-one","")),401,0,token)
        })
    }
    db,store:=t3RealStoreDB(t)
    t3Identity(t,store,"p1","w1","org-one","project-one")
    now:=time.Now().UTC()
    token:=t3Credential(t,store,"c1","p1",now.Add(-time.Hour),now.Add(time.Hour))
    body:=t3JSON("org-one","project-one","")
    t3Check(t,db,store,t3Req("POST",token,body+" {}"),400,0,token)
    t3Check(t,db,store,t3Req("DELETE",token,""),405,0,token)
    duplicate:=t3Req("POST",token,body)
    duplicate.Header.Add("Authorization","Bearer "+token)
    t3Check(t,db,store,duplicate,401,0,token)
    for _,tc:=range []struct{size,status,calls int}{
        {middleware.DubBridgeS2SMaxRequestBytes,202,1},
        {middleware.DubBridgeS2SMaxRequestBytes+1,413,0},
    }{
        long:=t3JSON("org-one","project-one",strings.Repeat("x",tc.size-len(body)))
        if len(long)!=tc.size{t.Fatalf("fixture len %d vs %d",len(long),tc.size)}
        t3Check(t,db,store,t3Req("POST",token,long),tc.status,tc.calls,token)
    }
    t.Setenv("JWT_SECRET","t3-test-only-secret")
    human,err:=pkgauth.GenerateJWT("human","w1")
    if err!=nil{t.Fatal(err)}
    t3Check(t,db,store,t3Req("POST",human,body),401,0,human)
    hits:=0
    w:=httptest.NewRecorder()
    middleware.AuthMiddleware(http.HandlerFunc(func(http.ResponseWriter,*http.Request){hits++})).ServeHTTP(w,t3Req("GET",token,""))
    if w.Code!=401||hits!=0{t.Fatal("service credential accepted by human JWT")}
}

func TestB2A3T3SQLiteMalformedSourceAndClosedStore(t *testing.T) {
	db, store := t3RealStoreDB(t)
	t3Identity(t, store, "p1", "w1", "org-one", "project-one")
	now := time.Now().UTC()
	bearer := t3Credential(t, store, "c1", "p1", now.Add(-time.Hour), now.Add(time.Hour))
	body := t3JSON("org-one", "project-one", "")

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		t.Fatal(err)
	}
	delete(envelope, "source")
	missingSource, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	envelope["source"] = json.RawMessage("null")
	nullSource, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}

	// A duplicated top-level assertion must not be interpreted differently
	// by the guard and its future downstream envelope consumer.
	duplicateSource := strings.TrimSuffix(body, "}") +
		`,"source":{"product":"dubbridge","org_id":"org-one","project_id":"project-one"}}`
	duplicateOrg := `{"contract_version":"1","operation_id":"operation_123456789","capability_key":"integration.diagnostic","input_schema_version":"1","trace_id":"trace_12345678","source":{"product":"dubbridge","org_id":"org-one","org_id":"org-two","project_id":"project-one"},"input":{}}`

	for _, payload := range []string{
		string(missingSource),
		string(nullSource),
		duplicateSource,
		duplicateOrg,
		`{"source":true}`,
	} {
		t3Check(t, db, store, t3Req("POST", bearer, payload), 400, 0, bearer)
	}

	wrongType := t3Req("POST", bearer, body)
	wrongType.Header.Set("Content-Type", "text/plain")
	t3Check(t, db, store, wrongType, 415, 0, bearer)

	// Closed storage must deny before the downstream handler, without a panic
	// or leaking an opaque bearer into the response body.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	spy := &t3Downstream{}
	w := httptest.NewRecorder()
	middleware.DubBridgeServiceAuth(store, nil, spy).ServeHTTP(
		w, t3Req("POST", bearer, body),
	)
	if w.Code != http.StatusServiceUnavailable || spy.calls != 0 {
		t.Fatalf("closed SQLite returned status=%d, calls=%d", w.Code, spy.calls)
	}
	if strings.Contains(w.Body.String(), bearer) {
		t.Fatal("closed SQLite disclosed the service bearer")
	}
}
