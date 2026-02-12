// Copyright 2026 Benjamin Toso <benjamin.toso@gmail.com>
// Licensed under the Apache License, Version 2.0

package main

import "net/http"

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>IMF — Immutable File Container</title>
<style>
:root{--bg:#0f1117;--surface:#1a1d27;--surface2:#232735;--surface3:#2a2e3f;--border:#2e3345;--border-light:#3a3f55;--text:#e1e4ed;--text-dim:#8b90a0;--text-faint:#5a5f70;--accent:#4f8ff7;--accent-glow:rgba(79,143,247,.12);--accent-strong:rgba(79,143,247,.25);--success:#34d399;--success-bg:rgba(52,211,153,.1);--error:#f87171;--error-bg:rgba(248,113,113,.1);--warning:#fbbf24;--warning-bg:rgba(251,191,36,.1);--radius:10px;--mono:'SF Mono','Fira Code','Consolas',monospace}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'SF Pro Display',-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:var(--bg);color:var(--text);min-height:100vh;line-height:1.6;overflow:hidden}
#launchScreen{display:flex;flex-direction:column;align-items:center;justify-content:center;height:100vh;gap:48px}
.launch-logo{text-align:center}
.launch-logo h1{font-size:42px;font-weight:700;letter-spacing:-1px;margin-bottom:8px}
.launch-logo h1 span{color:var(--accent)}
.launch-logo p{color:var(--text-dim);font-size:15px}
.launch-actions{display:flex;gap:24px}
.launch-card{width:240px;padding:36px 28px;background:var(--surface);border:1px solid var(--border);border-radius:16px;text-align:center;cursor:pointer;transition:all .25s}
.launch-card:hover{border-color:var(--accent);background:var(--accent-glow);transform:translateY(-4px);box-shadow:0 8px 32px rgba(79,143,247,.15)}
.launch-card .icon{font-size:48px;margin-bottom:16px}
.launch-card h3{font-size:16px;font-weight:600;margin-bottom:6px}
.launch-card p{font-size:13px;color:var(--text-dim)}
.launch-card input[type="file"]{display:none}
.launch-key-section{display:flex;align-items:center;gap:12px;padding:12px 20px;background:var(--surface);border:1px solid var(--border);border-radius:10px}
.launch-key-section .status{font-size:13px;color:var(--text-dim)}
.launch-key-section .status.loaded{color:var(--success)}
.lkb{padding:6px 16px;border-radius:6px;font-size:12px;font-weight:500;cursor:pointer;border:1px solid var(--border);background:var(--surface2);color:var(--text);transition:all .2s}
.lkb:hover{border-color:var(--accent);color:var(--accent)}
.modal-overlay{display:none;position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,.6);z-index:100;align-items:center;justify-content:center}
.modal-overlay.active{display:flex}
.modal{background:var(--surface);border:1px solid var(--border);border-radius:16px;padding:32px;width:420px;box-shadow:0 16px 64px rgba(0,0,0,.5)}
.modal h2{font-size:18px;margin-bottom:20px}
.modal label{display:block;font-size:13px;color:var(--text-dim);font-weight:500;margin-bottom:6px}
.modal input[type="text"],.modal input[type="password"],.modal input[type="date"]{width:100%;padding:10px 14px;background:var(--bg);border:1px solid var(--border);border-radius:8px;color:var(--text);font-size:14px;outline:none;margin-bottom:16px}
.modal input:focus{border-color:var(--accent)}
.modal-btns{display:flex;gap:12px;justify-content:flex-end;margin-top:8px}
.seal-check{display:flex;align-items:center;gap:8px;font-size:13px;margin-bottom:12px}
.seal-check input{accent-color:var(--accent)}
#workspace{display:none;height:100vh;flex-direction:column}
#workspace.active{display:flex}
.titlebar{display:flex;align-items:center;justify-content:space-between;padding:10px 20px;background:var(--surface);border-bottom:1px solid var(--border)}
.titlebar-left{display:flex;align-items:center;gap:12px}
.back-btn{padding:4px 12px;border-radius:6px;border:1px solid var(--border);background:transparent;color:var(--text-dim);cursor:pointer;font-size:14px}
.back-btn:hover{border-color:var(--accent);color:var(--accent)}
.container-name{font-size:15px;font-weight:600}
.state-badge{padding:3px 10px;border-radius:12px;font-size:11px;font-weight:600;text-transform:uppercase;letter-spacing:.5px}
.state-badge.open{background:var(--warning-bg);color:var(--warning);border:1px solid var(--warning)}
.state-badge.sealed{background:var(--success-bg);color:var(--success);border:1px solid var(--success)}
.titlebar-actions{display:flex;gap:8px}
.workspace-body{display:flex;flex:1;overflow:hidden}
.sidebar{width:260px;background:var(--surface);border-right:1px solid var(--border);overflow-y:auto}
.sidebar-section{padding:16px 20px;border-bottom:1px solid var(--border)}
.sidebar-section h4{font-size:11px;text-transform:uppercase;letter-spacing:.8px;color:var(--text-faint);margin-bottom:12px}
.meta-row{display:flex;justify-content:space-between;font-size:13px;margin-bottom:8px}
.meta-row .label{color:var(--text-dim)}
.meta-row .value{font-weight:500;text-align:right}
.meta-row .value.good{color:var(--success)}
.meta-row .value.warn{color:var(--warning)}
.meta-row .value.bad{color:var(--error)}
.verify-status{padding:10px;border-radius:8px;text-align:center;font-size:13px;font-weight:600;margin-top:8px}
.verify-status.pass{background:var(--success-bg);color:var(--success)}
.verify-status.fail{background:var(--error-bg);color:var(--error)}
.verify-status.pending{background:var(--surface2);color:var(--text-dim)}
.file-area{flex:1;display:flex;flex-direction:column;overflow:hidden;position:relative}
.file-toolbar{display:flex;align-items:center;justify-content:space-between;padding:10px 20px;border-bottom:1px solid var(--border);background:var(--surface2)}
.file-toolbar .info{font-size:13px;color:var(--text-dim)}
.tb{padding:6px 14px;border-radius:6px;font-size:12px;font-weight:500;cursor:pointer;border:1px solid var(--border);background:transparent;color:var(--text);transition:all .2s;text-decoration:none;display:inline-block}
.tb:hover{border-color:var(--accent);color:var(--accent)}
.tb.primary{background:var(--accent);color:#fff;border-color:var(--accent)}
.tb.primary:hover{background:#3d7de5}
.tb.success{background:var(--success);color:var(--bg);border-color:var(--success)}
.file-list-header{display:grid;grid-template-columns:32px 1fr 90px 90px 80px;gap:8px;padding:8px 20px;font-size:11px;font-weight:600;color:var(--text-faint);text-transform:uppercase;letter-spacing:.5px;border-bottom:1px solid var(--border);background:var(--surface)}
.file-scroll{flex:1;overflow-y:auto}
.frow{display:grid;grid-template-columns:32px 1fr 90px 90px 80px;gap:8px;padding:10px 20px;font-size:13px;border-bottom:1px solid var(--border);cursor:default;transition:background .12s;align-items:center}
.frow:hover{background:var(--accent-glow)}
.frow.selected{background:var(--accent-strong)}
.frow .icon{font-size:20px;text-align:center}
.frow .fname{font-weight:500;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.frow .fsize,.frow .ftype{color:var(--text-dim);font-size:12px}
.frow .factions{display:flex;gap:4px}
.fa-btn{padding:3px 8px;border-radius:4px;border:1px solid var(--border);background:transparent;color:var(--text-dim);font-size:11px;cursor:pointer;transition:all .15s}
.fa-btn:hover{border-color:var(--accent);color:var(--accent)}
.empty-state{flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;color:var(--text-dim);gap:16px}
.empty-state .icon{font-size:64px;opacity:.4}
.empty-state .hint{font-size:13px;color:var(--text-faint)}
.drop-overlay{display:none;position:absolute;top:0;left:0;right:0;bottom:0;background:rgba(79,143,247,.08);border:3px dashed var(--accent);z-index:50;align-items:center;justify-content:center;font-size:18px;font-weight:600;color:var(--accent)}
.drop-overlay.active{display:flex}
.preview-pane{display:none;width:320px;background:var(--surface);border-left:1px solid var(--border);overflow-y:auto;flex-direction:column}
.preview-pane.active{display:flex}
.preview-top{padding:16px;border-bottom:1px solid var(--border);text-align:center}
.preview-thumb{background:var(--bg);border-radius:8px;overflow:hidden;margin-bottom:12px;min-height:160px;display:flex;align-items:center;justify-content:center}
.preview-thumb img{max-width:100%;max-height:200px}
.preview-thumb pre{padding:12px;font-size:10px;font-family:var(--mono);max-height:200px;overflow:auto;text-align:left;width:100%;color:var(--text);margin:0}
.preview-thumb iframe{width:100%;height:200px;border:none}
.preview-thumb .big-icon{font-size:64px;opacity:.5;padding:32px}
.pv-name{font-size:14px;font-weight:600}
.pv-meta{padding:16px;font-size:12px}
.pv-meta-row{display:flex;justify-content:space-between;padding:6px 0;border-bottom:1px solid var(--border)}
.pv-meta-row .label{color:var(--text-dim)}
.pv-actions{padding:16px;display:flex;flex-direction:column;gap:8px}
.btn{padding:10px 24px;border-radius:8px;font-size:14px;font-weight:500;cursor:pointer;border:none;transition:all .2s}
.btn-primary{background:var(--accent);color:#fff}
.btn-primary:hover{background:#3d7de5}
.btn-secondary{background:var(--surface2);color:var(--text);border:1px solid var(--border)}
.btn-secondary:hover{border-color:var(--accent)}
.toast{position:fixed;bottom:24px;right:24px;padding:12px 20px;border-radius:var(--radius);font-size:13px;font-weight:500;animation:slideIn .3s ease;z-index:200;max-width:400px}
.toast.success{background:var(--success-bg);color:var(--success);border:1px solid var(--success)}
.toast.error{background:var(--error-bg);color:var(--error);border:1px solid var(--error)}
@keyframes slideIn{from{transform:translateY(20px);opacity:0}}
</style>
</head>
<body>
<div id="launchScreen">
  <div class="launch-logo"><h1><span>IMF</span></h1><p>Immutable File Container</p></div>
  <div class="launch-actions">
    <div class="launch-card" onclick="document.getElementById('openFile').click()">
      <div class="icon">&#128194;</div><h3>Open Existing</h3><p>Open and inspect an .imf container</p>
      <input type="file" id="openFile" accept=".imf" onchange="handleOpen(this.files[0])">
    </div>
    <div class="launch-card" onclick="showModal('createModal')">
      <div class="icon">&#10010;</div><h3>Create New</h3><p>Create a new container and add files</p>
    </div>
  </div>
  <div class="launch-key-section">
    <span id="keyStatus" class="status">No signing key loaded</span>
    <button class="lkb" onclick="doKeygen()">Generate Key</button>
    <button class="lkb" onclick="document.getElementById('keyFile').click()">Load Key</button>
    <input type="file" id="keyFile" accept=".pem" style="display:none" onchange="doLoadKey(this.files[0])">
  </div>
</div>

<div class="modal-overlay" id="createModal">
  <div class="modal">
    <h2>Create New Container</h2>
    <label>Container Name</label>
    <input type="text" id="createName" placeholder="my-archive">
    <div class="modal-btns">
      <button class="btn btn-secondary" onclick="hideModal('createModal')">Cancel</button>
      <button class="btn btn-primary" onclick="doCreate()">Create</button>
    </div>
  </div>
</div>

<div class="modal-overlay" id="sealModal">
  <div class="modal">
    <h2>Seal Container</h2>
    <p style="font-size:13px;color:var(--text-dim);margin-bottom:20px">Once sealed, no files can be added or modified. This is permanent.</p>
    <div class="seal-check"><input type="checkbox" id="sealEmbed" checked><label for="sealEmbed">Embed public key (self-verifying)</label></div>
    <label>Encryption Passphrase (optional)</label>
    <input type="password" id="sealPass" placeholder="Leave blank to skip encryption">
    <label>Expiration Date (optional)</label>
    <input type="date" id="sealExp">
    <div class="modal-btns">
      <button class="btn btn-secondary" onclick="hideModal('sealModal')">Cancel</button>
      <button class="btn btn-primary" onclick="doSeal()">Seal Forever</button>
    </div>
  </div>
</div>

<div id="workspace">
  <div class="titlebar">
    <div class="titlebar-left">
      <button class="back-btn" onclick="goHome()">&#8592;</button>
      <span class="container-name" id="wsName"></span>
      <span class="state-badge" id="wsBadge"></span>
    </div>
    <div class="titlebar-actions" id="wsActions"></div>
  </div>
  <div id="locBar" style="padding:4px 20px;background:var(--bg);border-bottom:1px solid var(--border);font-size:11px;color:var(--text-faint);display:none">
    &#128193; Saved at: <span id="locPath"></span>
  </div>
  <div class="workspace-body">
    <div class="sidebar">
      <div class="sidebar-section" id="sMeta"></div>
      <div class="sidebar-section" id="sVerify"></div>
      <div class="sidebar-section" id="sCrypto"></div>
      <div class="sidebar-section" id="sAnchor"></div>
    </div>
    <div class="file-area" id="fileArea">
      <div class="file-toolbar" id="fileTB"></div>
      <div class="file-list-header" id="flHead"><div></div><div>Name</div><div>Size</div><div>Type</div><div></div></div>
      <div class="file-scroll" id="fileScroll"></div>
      <div class="drop-overlay" id="dropOverlay">Drop files to add</div>
    </div>
    <div class="preview-pane" id="pvPane">
      <div class="preview-top"><div class="preview-thumb" id="pvThumb"></div><div class="pv-name" id="pvName"></div></div>
      <div class="pv-meta" id="pvMeta"></div>
      <div class="pv-actions" id="pvAct"></div>
    </div>
  </div>
</div>

<script>
let cName='',cState='',cInfo=null,files=[],selIdx=-1;

// Launch
async function handleOpen(file){
  if(!file)return;
  const f=new FormData();f.append('container_file',file);
  // Upload container to server
  const f2=new FormData();f2.append('container_file',file);
  await fetch('/api/upload-container',{method:'POST',body:f2});
  // Get info
  const f3=new FormData();f3.append('container',file.name);
  const r=await(await fetch('/api/info',{method:'POST',body:f3})).json();
  if(!r.success){toast(r.error,'error');return}
  cName=file.name;cInfo=r.data;cState=cInfo.State;
  // If sealed, extract for preview
  if(cState==='sealed'){
    const ef=new FormData();ef.append('container',cName);ef.append('passphrase','');ef.append('ignore_expiry','true');
    await fetch('/api/extract',{method:'POST',body:ef});
  }
  enterWS();
}

function showModal(id){document.getElementById(id).classList.add('active')}
function hideModal(id){document.getElementById(id).classList.remove('active')}

async function doCreate(){
  const name=document.getElementById('createName').value.trim()||'container';
  const r=await pf('/api/create',{name});
  if(r.success){
    cName=r.data.name;cState='open';
    cInfo={State:'open',CreatedAt:new Date().toISOString(),FileCount:0,Encrypted:false,HasPubKey:false};
    hideModal('createModal');enterWS();
  }else toast(r.error,'error');
}

async function doKeygen(){
  const r=await pf('/api/keygen',{});
  if(r.success){toast('Key pair generated','success');setKey(true,'Private key loaded')}
  else toast(r.error,'error');
}
async function doLoadKey(file){
  if(!file)return;
  const f=new FormData();f.append('key',file);
  const r=await(await fetch('/api/load-key',{method:'POST',body:f})).json();
  if(r.success){toast(r.message,'success');setKey(true,r.message)}
  else toast(r.error,'error');
}
function setKey(ok,txt){const e=document.getElementById('keyStatus');e.textContent=txt;e.className='status'+(ok?' loaded':'')}

// Workspace
async function enterWS(){
  document.getElementById('launchScreen').style.display='none';
  document.getElementById('workspace').classList.add('active');
  // Show the file location bar
  try{
    const wd=await(await fetch('/api/workdir')).json();
    if(wd.success){
      document.getElementById('locPath').textContent=wd.data.path+'/'+cName;
      document.getElementById('locBar').style.display='';
    }
  }catch(e){}
  renderWS();await refreshFiles();
  if(cState==='sealed')autoVerify();
}
function goHome(){
  document.getElementById('workspace').classList.remove('active');
  document.getElementById('launchScreen').style.display='';
  cName='';cState='';cInfo=null;files=[];selIdx=-1;
  document.getElementById('pvPane').classList.remove('active');
}

function renderWS(){
  document.getElementById('wsName').textContent=cName;
  const b=document.getElementById('wsBadge');b.textContent=cState;b.className='state-badge '+cState;
  const a=document.getElementById('wsActions');
  if(cState==='open'){
    a.innerHTML='<button class="tb" onclick="document.getElementById(\'addIn\').click()">+ Add Files</button>'+
      '<button class="tb primary" onclick="showModal(\'sealModal\')">Seal</button>'+
      '<input type="file" id="addIn" multiple style="display:none" onchange="addF(this.files)">';
  }else{
    a.innerHTML='<a href="/api/download?file='+encodeURIComponent(cName)+'" class="tb">Download .imf</a>'+
      '<button class="tb" onclick="anchorContainer()" style="background:var(--warning-bg);color:var(--warning);border-color:var(--warning)">&#9875; Anchor to Bitcoin</button>'+
      '<button class="tb success" onclick="extractDL()">Extract All</button>';
  }
  renderSB();
  document.getElementById('fileTB').innerHTML='<div class="info" id="fCount"></div>'+
    (cState==='sealed'?'<a href="/api/download-zip" class="tb success" style="font-size:11px;padding:5px 12px">Download All</a>':'');
  if(cState==='open')setupDrop();
}

function renderSB(){
  const cr=cInfo.CreatedAt?new Date(cInfo.CreatedAt).toLocaleString():'—';
  const se=cInfo.SealedAt?new Date(cInfo.SealedAt).toLocaleString():'—';
  let ex='None',ec='';
  if(cInfo.ExpiresAt){ex=new Date(cInfo.ExpiresAt).toLocaleDateString();ec=cInfo.Expired?'bad':'good';if(cInfo.Expired)ex+=' (EXPIRED)'}
  document.getElementById('sMeta').innerHTML='<h4>Container</h4>'+
    mr('State',cState.toUpperCase(),cState==='sealed'?'good':'warn')+
    mr('Created',cr)+(cState==='sealed'?mr('Sealed',se):'')+
    mr('Expires',ex,ec)+mr('Files',cInfo.FileCount||0);
  document.getElementById('sCrypto').innerHTML='<h4>Security</h4>'+
    mr('Encrypted',cInfo.Encrypted?'Yes':'No',cInfo.Encrypted?'good':'')+
    mr('Pub Key',cInfo.HasPubKey?'Embedded':'None',cInfo.HasPubKey?'good':'');
  document.getElementById('sVerify').innerHTML='<h4>Integrity</h4>'+
    '<div class="verify-status pending" id="vBadge">'+(cState==='sealed'?'Checking...':'Not yet sealed')+'</div>';
  // Show blockchain anchor section for sealed containers
  const aDiv=document.getElementById('sAnchor');
  if(cState==='sealed'){
    aDiv.innerHTML='<h4>Blockchain Anchor</h4>'+
      '<div style="font-size:12px;color:var(--text-dim)" id="anchorStatus">Checking...</div>';
    // Check if an .ots proof already exists for this container
    checkAnchorStatus();
  }else{aDiv.innerHTML='';}
}

function mr(l,v,c){return'<div class="meta-row"><span class="label">'+l+'</span><span class="value'+(c?' '+c:'')+'">'+v+'</span></div>'}

// Files
async function refreshFiles(){
  const f=new FormData();f.append('container',cName);
  const r=await(await fetch('/api/list',{method:'POST',body:f})).json();
  files=(r.success&&r.data)?r.data:[];
  renderFL();
}

function renderFL(){
  document.getElementById('fCount').textContent=files.length+' item'+(files.length!==1?'s':'');
  const s=document.getElementById('fileScroll');
  if(!files.length){
    document.getElementById('flHead').style.display='none';
    s.innerHTML='<div class="empty-state"><div class="icon">'+(cState==='open'?'&#128194;':'&#128274;')+'</div>'+
      '<p>'+(cState==='open'?'No files yet':'Empty container')+'</p>'+
      (cState==='open'?'<div class="hint">Drag and drop files here or click + Add Files</div>':'')+
    '</div>';return;
  }
  document.getElementById('flHead').style.display='';
  s.innerHTML=files.map((f,i)=>{
    const ext=f.OriginalName.split('.').pop().toLowerCase();
    const t=cType(ext);
    return'<div class="frow'+(i===selIdx?' selected':'')+'" onclick="sel('+i+')" ondblclick="openF('+i+')">'+
      '<div class="icon">'+ico(t)+'</div>'+
      '<div class="fname">'+f.OriginalName+'</div>'+
      '<div class="fsize">'+fmtS(f.OriginalSize)+'</div>'+
      '<div class="ftype">'+ext.toUpperCase()+'</div>'+
      '<div class="factions">'+
        (cState==='sealed'?'<button class="fa-btn" onclick="event.stopPropagation();openF('+i+')">Open</button>'+
          '<button class="fa-btn" onclick="event.stopPropagation();saveF('+i+')">Save</button>':'')+
      '</div></div>';
  }).join('');
}

function sel(i){selIdx=i;renderFL();showPV(files[i])}

function showPV(f){
  document.getElementById('pvPane').classList.add('active');
  const ext=f.OriginalName.split('.').pop().toLowerCase();
  const t=cType(ext);
  const url='/api/serve-file?file='+encodeURIComponent(f.OriginalName);
  document.getElementById('pvName').textContent=f.OriginalName;
  const th=document.getElementById('pvThumb');
  if(cState==='sealed'){
    if(['jpg','jpeg','png','gif','webp','svg','bmp'].includes(ext))th.innerHTML='<img src="'+url+'">';
    else if(ext==='pdf')th.innerHTML='<iframe src="'+url+'"></iframe>';
    else if(['txt','md','csv','log','json','xml','yaml','yml','go','py','js','html','css','sh','toml'].includes(ext)){
      fetch(url).then(r=>r.text()).then(text=>{
        th.innerHTML='<pre>'+text.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').substring(0,5000)+'</pre>'});
    }else th.innerHTML='<div class="big-icon">'+ico(t)+'</div>';
  }else th.innerHTML='<div class="big-icon">'+ico(t)+'</div>';

  document.getElementById('pvMeta').innerHTML=
    pvr('Name',f.OriginalName)+pvr('Size',fmtS(f.OriginalSize))+pvr('Type',ext.toUpperCase())+
    pvr('SHA-256','<span style="font-family:var(--mono);font-size:10px;word-break:break-all">'+f.SHA256+'</span>');

  const a=document.getElementById('pvAct');
  if(cState==='sealed'){
    a.innerHTML='<button class="btn btn-primary" style="font-size:13px;padding:8px" onclick="openF('+selIdx+')">Open File</button>'+
      '<a href="/api/download?file='+encodeURIComponent(f.OriginalName)+'" class="btn btn-secondary" style="font-size:13px;padding:8px;text-decoration:none;text-align:center">Save to Disk</a>';
  }else a.innerHTML='<div style="font-size:12px;color:var(--text-dim);text-align:center">Seal the container to open or save files</div>';
}

function pvr(l,v){return'<div class="pv-meta-row"><span class="label">'+l+'</span><span>'+v+'</span></div>'}

// Actions
function openF(i){
  if(cState!=='sealed'){toast('Seal the container first','error');return}
  window.open('/api/serve-file?file='+encodeURIComponent(files[i].OriginalName),'_blank');
}
function saveF(i){window.location.href='/api/download?file='+encodeURIComponent(files[i].OriginalName)}

async function extractDL(){
  const pass=prompt('Decryption passphrase (blank if unencrypted):');
  if(pass===null)return;
  const f=new FormData();f.append('container',cName);f.append('passphrase',pass||'');
  const r=await(await fetch('/api/extract',{method:'POST',body:f})).json();
  if(r.success){toast('Downloading files...','success');setTimeout(()=>window.location.href='/api/download-zip',500)}
  else toast(r.error,'error');
}

// Add files
async function addF(fl){
  if(!fl.length)return;
  if(cState!=='open'){toast('Cannot add to sealed container','error');return}
  const f=new FormData();f.append('container',cName);
  for(const x of fl)f.append('files',x);
  const r=await(await fetch('/api/add',{method:'POST',body:f})).json();
  if(r.success){
    toast('Added '+fl.length+' file(s)','success');
    const f2=new FormData();f2.append('container',cName);
    const ir=await(await fetch('/api/info',{method:'POST',body:f2})).json();
    if(ir.success)cInfo=ir.data;
    renderSB();await refreshFiles();
  }else toast(r.error,'error');
}

function setupDrop(){
  const a=document.getElementById('fileArea'),o=document.getElementById('dropOverlay');
  let dc=0;
  a.ondragenter=e=>{e.preventDefault();dc++;o.classList.add('active')};
  a.ondragleave=()=>{dc--;if(dc<=0){o.classList.remove('active');dc=0}};
  a.ondragover=e=>e.preventDefault();
  a.ondrop=e=>{e.preventDefault();o.classList.remove('active');dc=0;if(e.dataTransfer.files.length)addF(e.dataTransfer.files)};
}

// Seal
async function doSeal(){
  if(!files.length){toast('Add files first','error');return}
  // Check if a signing key is loaded
  try{
    const ks=await(await fetch('/api/key-status')).json();
    if(!ks.data.loaded){
      // Show key prompt modal
      const gen=await showKeyPrompt();
      if(gen){
        const kr=await fetch('/api/keygen',{method:'POST'});
        const kd=await kr.json();
        if(kd.success){toast('Signing key auto-generated','success');}
        else{toast('Key generation failed: '+kd.error,'error');return;}
      }else{
        hideModal('sealModal');
        toast('A signing key is required to seal — redirecting to start','info');
        document.getElementById('workspace').classList.remove('active');
        document.getElementById('landing').classList.add('active');
        return;
      }
    }
  }catch(e){console.error('Key status check failed',e);}
  const r=await pf('/api/seal',{
    container:cName,passphrase:document.getElementById('sealPass').value,
    expires:document.getElementById('sealExp').value,
    embed_key:document.getElementById('sealEmbed').checked?'true':'false'
  });
  if(r.success){
    cState='sealed';hideModal('sealModal');toast('Container sealed','success');
    const f=new FormData();f.append('container',cName);
    const ir=await(await fetch('/api/info',{method:'POST',body:f})).json();
    if(ir.success)cInfo=ir.data;
    // Extract for preview
    const ef=new FormData();ef.append('container',cName);ef.append('passphrase',document.getElementById('sealPass').value);
    await fetch('/api/extract',{method:'POST',body:ef});
    renderWS();await refreshFiles();autoVerify();
  }else toast(r.error,'error');
}

// Key prompt modal
function showKeyPrompt(){
  return new Promise(resolve=>{
    const o=document.createElement('div');
    o.className='modal-overlay active';
    o.style.zIndex='1001';
    o.innerHTML='<div class="modal">'+
      '<h2>&#128273; No Signing Key</h2>'+
      '<p style="font-size:13px;color:var(--text-dim);margin-bottom:20px">'+
      'A signing key is required to seal the container. The key cryptographically signs the manifest to guarantee integrity.</p>'+
      '<p style="font-size:13px;color:var(--text-dim);margin-bottom:20px">'+
      'Would you like to generate an Ed25519 key pair automatically?</p>'+
      '<div class="modal-btns">'+
        '<button class="btn btn-secondary" id="kpNo">Go Back</button>'+
        '<button class="btn btn-primary" id="kpYes">&#10003; Generate Key</button>'+
      '</div></div>';
    document.body.appendChild(o);
    document.getElementById('kpYes').onclick=()=>{o.remove();resolve(true)};
    document.getElementById('kpNo').onclick=()=>{o.remove();resolve(false)};
  });
}

// Verify
async function autoVerify(){
  const f=new FormData();f.append('container',cName);
  const r=await(await fetch('/api/verify',{method:'POST',body:f})).json();
  const e=document.getElementById('vBadge');
  if(r.success){e.className='verify-status pass';e.innerHTML='&#10003; Verified'}
  else{e.className='verify-status fail';e.innerHTML='&#10007; '+r.error}
}

// Anchor to Bitcoin via OpenTimestamps
async function anchorContainer(){
  toast('Anchoring to Bitcoin via OpenTimestamps...','info');
  const f=new FormData();f.append('container',cName);
  const r=await(await fetch('/api/anchor',{method:'POST',body:f})).json();
  if(r.success){
    toast('Anchored to Bitcoin!','success');
    showAnchorResult(r.data);
  }else{
    toast('Anchor failed: '+r.error,'error');
  }
}

// Check if .ots proof exists and verify it
async function checkAnchorStatus(){
  const f=new FormData();f.append('container',cName);
  try{
    const r=await(await fetch('/api/anchor-verify',{method:'POST',body:f})).json();
    if(r.success){
      showAnchorVerified(r.data);
    }else{
      showAnchorNotFound();
    }
  }catch(e){showAnchorNotFound();}
}

// Show anchor result after submitting
function showAnchorResult(data){
  const aDiv=document.getElementById('sAnchor');
  if(!aDiv)return;
  aDiv.innerHTML='<h4>Blockchain Anchor</h4>'+
    mr('Status','Submitted','good')+
    mr('Hash',data.hash.substring(0,16)+'...')+
    mr('Server',data.server.replace('https://',''))+
    mr('Submitted',new Date(data.timestamp).toLocaleString())+
    '<div style="margin-top:10px;display:flex;flex-direction:column;gap:6px">'+
      '<a href="/api/download?file='+encodeURIComponent(cName+'.ots')+'" class="tb success" style="font-size:11px;padding:4px 10px;text-decoration:none;text-align:center">Download .ots proof</a>'+
      '<button class="tb" onclick="verifyAnchor()" style="font-size:11px;padding:4px 10px">Verify Anchor</button>'+
    '</div>';
}

// Show verified anchor status
function showAnchorVerified(data){
  const aDiv=document.getElementById('sAnchor');
  if(!aDiv)return;
  aDiv.innerHTML='<h4>Blockchain Anchor</h4>'+
    '<div class="verify-status pass" style="margin-bottom:10px">&#10003; Proof matches container</div>'+
    mr('Hash',data.hash.substring(0,16)+'...')+
    mr('Proof size',data.proof_size+' bytes')+
    '<div style="margin-top:10px;display:flex;flex-direction:column;gap:6px">'+
      '<a href="/api/download?file='+encodeURIComponent(cName+'.ots')+'" class="tb success" style="font-size:11px;padding:4px 10px;text-decoration:none;text-align:center">Download .ots proof</a>'+
      '<a href="https://opentimestamps.org" target="_blank" class="tb" style="font-size:11px;padding:4px 10px;text-decoration:none;text-align:center">Verify on Bitcoin &#8599;</a>'+
    '</div>'+
    '<div style="margin-top:8px;font-size:10px;color:var(--text-faint)">'+
      'Drop your .ots file at opentimestamps.org for full Bitcoin block verification.'+
    '</div>';
}

// Show when no anchor exists yet
function showAnchorNotFound(){
  const aDiv=document.getElementById('sAnchor');
  if(!aDiv)return;
  aDiv.innerHTML='<h4>Blockchain Anchor</h4>'+
    '<div style="font-size:12px;color:var(--text-dim)">Not yet anchored</div>'+
    '<div style="margin-top:6px;font-size:11px;color:var(--text-faint)">'+
      'Click "&#9875; Anchor to Bitcoin" above to timestamp this container on the blockchain.'+
    '</div>';
}

// Verify existing anchor
async function verifyAnchor(){
  toast('Verifying anchor proof...','info');
  const f=new FormData();f.append('container',cName);
  const r=await(await fetch('/api/anchor-verify',{method:'POST',body:f})).json();
  if(r.success){
    toast('Anchor verified — proof matches container','success');
    showAnchorVerified(r.data);
  }else{
    toast('Anchor verification failed: '+r.error,'error');
  }
}

// Helpers
async function pf(url,d){const f=new FormData();for(const[k,v]of Object.entries(d))f.append(k,v);return(await fetch(url,{method:'POST',body:f})).json()}
function toast(m,t){const e=document.createElement('div');e.className='toast '+t;e.textContent=m;document.body.appendChild(e);setTimeout(()=>e.remove(),4000)}
function fmtS(b){if(b<1024)return b+' B';if(b<1048576)return(b/1024).toFixed(1)+' KB';return(b/1048576).toFixed(1)+' MB'}
function ico(t){return{image:'&#128444;',pdf:'&#128196;',text:'&#128196;',code:'&#128187;',document:'&#128203;',archive:'&#128230;',audio:'&#127925;',video:'&#127909;',other:'&#128196;'}[t]||'&#128196;'}
function cType(e){
  if(['jpg','jpeg','png','gif','webp','svg','bmp','ico'].includes(e))return'image';
  if(e==='pdf')return'pdf';
  if(['txt','md','csv','log','json','xml','yaml','yml','toml'].includes(e))return'text';
  if(['go','py','js','ts','java','c','cpp','h','rs','rb','sh','html','css'].includes(e))return'code';
  if(['doc','docx','xls','xlsx','ppt','pptx'].includes(e))return'document';
  if(['zip','tar','gz','7z','rar','imf'].includes(e))return'archive';
  return'other';
}

// Keyboard
document.addEventListener('keydown',e=>{
  if(!document.getElementById('workspace').classList.contains('active'))return;
  if(e.key==='ArrowDown'){e.preventDefault();sel(Math.min(selIdx+1,files.length-1))}
  if(e.key==='ArrowUp'){e.preventDefault();sel(Math.max(selIdx-1,0))}
  if(e.key==='Enter'&&selIdx>=0){e.preventDefault();openF(selIdx)}
  if(e.key==='Escape'){document.getElementById('pvPane').classList.remove('active');selIdx=-1;renderFL()}
});
document.getElementById('createName').addEventListener('keydown',e=>{if(e.key==='Enter')doCreate()});
</script>
</body>
</html>` + "`"
