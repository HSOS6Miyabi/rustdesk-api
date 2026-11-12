package admin

import "testing"

func TestParseRustdeskReply(t *testing.T) {
    cases:=[]struct{
        name,target,command,option,reply,kind string
        active bool
        count int
        value float64
    }{
        {name:"relay state",target:"21115",command:"always-use-relay",reply:"ALWAYS_USE_RELAY: true\n",kind:"boolean",active:true},
        {name:"must login",target:"21115",command:"must-login",reply:"MUST_LOGIN: false\n",kind:"boolean"},
        {name:"relay addresses",target:"21115",command:"relay-servers",reply:"relay1:21117\nrelay2:21117\n",kind:"list",count:2},
        {name:"empty relay",target:"21115",command:"relay-servers",reply:"",kind:"list"},
        {name:"blocklist true",target:"21117",command:"blocklist",option:"192.0.2.1",reply:"true\n",kind:"boolean",active:true},
        {name:"blacklist false",target:"21117",command:"blacklist",option:"192.0.2.1",reply:"false\n",kind:"boolean"},
        {name:"read blacklist",target:"21117",command:"blacklist",reply:"192.0.2.1\n198.51.100.3\n",kind:"list",count:2},
        {name:"limit",target:"21117",command:"limit-speed",reply:"1.25Mb/s\n",kind:"metric",value:1.25},
        {name:"check seconds",target:"21117",command:"downgrade-start-check",reply:"20s\n",kind:"metric",value:20},
        {name:"IP changes",target:"21115",command:"ip-changes",reply:"3\n123456: 9s [\"1.2.3.4\"]\n",kind:"table"},
        {name:"usage",target:"21117",command:"usage",reply:"192.0.2.1: 60s 15.12MB 50kb/s 30kb/s 10kb/s\n",kind:"usage",count:1},
        {name:"punch",target:"21115",command:"punch-requests",reply:"2026-11-12T00:00:00Z 1.2.3.4 -> 123@5.6.7.8\n",kind:"list",count:1},
        {name:"write accepted",target:"21117",command:"limit-speed",option:"3",reply:"",kind:"notice"},
        {name:"empty reply",target:"21117",command:"usage",reply:"",kind:"usage"},
    }
    for _,tc:=range cases {
        t.Run(tc.name,func(t *testing.T){
            got:=parseRustdeskReply(tc.target,tc.command,tc.option,tc.reply)
            if got["kind"]!=tc.kind {t.Fatalf("got %v kind, expected %q",got["kind"],tc.kind)}
            if tc.kind=="boolean" && got["active"]!=tc.active {t.Errorf("active got %v",got["active"])}
            if tc.kind=="metric" && got["value"]!=tc.value {t.Errorf("value got %v",got["value"])}
            if tc.kind=="list"||tc.kind=="usage" {
                if got["count"]!=tc.count {t.Errorf("count got %v, want %v",got["count"],tc.count)}
            }
            if got["summary"]=="" {t.Error("missing user-facing summary")}
        })
    }
}
