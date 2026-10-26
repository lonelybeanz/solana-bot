import json,sys,urllib.request
repo,tok=sys.argv[1],sys.argv[2]
from collections import Counter
c=Counter(); n=0
hdr={"Authorization":"Bearer "+tok,"Accept":"application/vnd.github+json"}
for p in (1,2,3):
    try: d=json.load(open(f"/tmp/e02/runs{p}.json"))
    except Exception: continue
    for r in d.get("workflow_runs",[])[:100]:
        n+=1
        try:
            q=urllib.request.Request(f"https://api.github.com/repos/{repo}/actions/runs/{r['id']}/jobs?per_page=100",headers=hdr)
            j=json.load(urllib.request.urlopen(q,timeout=20))
        except Exception as e: c[("ERR",str(e)[:40])]+=1; continue
        for job in j.get("jobs",[]):
            c[(job.get("runner_name"),",".join(job.get("labels") or []))]+=1
print("runs_scanned",n)
for k,v in c.most_common(): print("runner",k[0],"labels",k[1],"jobs",v)
