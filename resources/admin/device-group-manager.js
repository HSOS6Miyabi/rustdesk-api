"use strict";

// Device-group editing is deliberately isolated from the generic record editor.
// Device selection uses database row IDs, so changes target exact registered records.
window.RustDeskDeviceGroupManager = function createDeviceGroupManager(deps) {
  const {state,request,render,modal,toast,esc,dateTime,deviceStatusDot,referenceText,canPageAction} = deps;
  let searchTimer = null;
  const pageSize = 20;
  const manager = () => state.modal?.kind === "device-group-devices" ? state.modal : null;
  const currentPageCount = m => Math.max(1,Math.ceil(m.total / pageSize));

  function rerenderPreservingSearch() {
    const focused = document.activeElement?.id === "device-group-query";
    const start = focused ? document.activeElement.selectionStart : null;
    render();
    if (focused) {
      const field = document.getElementById("device-group-query");
      if (field) {
        field.focus();
        if (Number.isInteger(start)) field.setSelectionRange(Math.min(start,field.value.length),Math.min(start,field.value.length));
      }
    }
  }

  async function load(m) {
    if (!m || manager() !== m) return;
    const serial = (m.serial || 0) + 1;
    m.serial = serial;
    m.loading = true;
    m.error = "";
    rerenderPreservingSearch();
    const params = new URLSearchParams({page:String(m.page),page_size:String(pageSize)});
    if (m.query) params.set("keyword",m.query);
    const path = m.mode === "members"
      ? "/device_group/devices?group_id="+encodeURIComponent(m.group.id)+"&"+params
      : "/peer/list?"+params;
    try {
      const result = await request(path);
      if (manager() !== m || m.serial !== serial) return;
      m.rows = result.list || [];
      m.total = Number(result.total) || 0;
    } catch (error) {
      if (manager() !== m || m.serial !== serial) return;
      m.rows = [];
      m.total = 0;
      m.error = error.message;
    }
    if (manager() !== m || m.serial !== serial) return;
    m.loading = false;
    rerenderPreservingSearch();
  }

  async function open(id) {
    if (!canPageAction("devicegroups","view") || !canPageAction("devices","view")) {
      toast("当前账户无权查看设备列表",true);
      return;
    }
    const group = state.rows.find(row => String(row.id) === String(id));
    if (!group) { toast("未找到设备分组，请刷新页面",true); return; }
    const m = {kind:"device-group-devices",group,mode:"members",page:1,query:"",
      rows:[],total:0,loading:true,error:"",busy:false,serial:0,
      selected:new Set(),selectedDetails:new Map()};
    state.modal = m;
    render();
    await load(m);
  }

  function renderModal(m) {
    const adding = m.mode === "add";
    const canEdit = canPageAction("devicegroups","edit");
    const name = String(m.group.name || m.group.id);
    const switcher = '<div class="device-group-tabs" role="group" aria-label="设备分组管理方式">'+
      '<button type="button" class="btn '+(!adding?'btn-primary':'')+'" data-action="device-group-tab" data-mode="members" aria-pressed="'+!adding+'">组内设备</button>'+
      (canEdit?'<button type="button" class="btn '+(adding?'btn-primary':'')+'" data-action="device-group-tab" data-mode="add" aria-pressed="'+adding+'">添加设备</button>':'')+'</div>';
    const controls = '<div class="device-group-toolbar"><label class="device-group-search-label" for="device-group-query">'+
      (adding?'搜索已登记设备':'搜索组内设备')+'</label><div class="device-group-search-row">'+
      '<input id="device-group-query" type="search" autocomplete="off" value="'+esc(m.query)+'" placeholder="设备 ID、名称、主机名、用户或系统" aria-label="搜索设备">'+
      '<button type="button" class="btn" data-action="device-group-search">搜索</button></div></div>';
    const selectedCount = m.selected.size;
    const allSelected = m.rows.length > 0 && m.rows.every(row => {
      const selectable = !adding || Number(row.group_id) !== Number(m.group.id);
      return !selectable || m.selected.has(Number(row.row_id));
    });
    const checkboxHead = canEdit?'<th class="device-group-check"><input type="checkbox" aria-label="选择本页设备" data-action="device-group-select-all" '+
      (allSelected?'checked':'')+' '+(m.loading||m.busy?'disabled':'')+'></th>':'';
    const heading = '<div class="device-group-result-head"><strong>'+(adding?'可添加的设备':'当前组内设备')+'</strong><span>'+ (m.loading?'正在读取…':'共 '+m.total+' 台')+' · 已选 '+selectedCount+' 台</span></div>';
    const tableRows = (m.rows||[]).map(row => {
      const rowID = Number(row.row_id),isMember = Number(row.group_id) === Number(m.group.id);
      const disabled = m.loading || m.busy || (adding && isMember);
      const nameText = String(row.alias||row.hostname||row.id||'未命名设备');
      const groupName = row.group_id ? (isMember?name:referenceText('device_group',row.group_id)) : '未分组';
      return '<tr>'+(canEdit?'<td class="device-group-check"><input type="checkbox" class="device-group-selected" value="'+esc(rowID)+'" '+
        (m.selected.has(rowID)?'checked ':'')+(disabled?'disabled ':'')+'aria-label="选择设备 '+esc(row.id)+'"></td>':'')+
        '<td><div class="device-group-identity"><span>'+deviceStatusDot(row.id)+esc(nameText)+'</span><small>'+esc(row.id)+'</small></div></td>'+
        '<td>'+esc(row.os||'—')+'</td>'+
        '<td>'+esc(row.username||'—')+'</td>'+
        '<td><span class="device-group-current'+(adding&&!isMember&&row.group_id?' moving':'')+'">'+esc(groupName)+'</span></td>'+
        '<td>'+dateTime(row.last_online_time)+'</td></tr>';
    }).join('');
    const body = '<div class="device-group-editor">'+switcher+controls+heading+
      (m.error?'<div class="device-group-error" role="alert">'+esc(m.error)+'</div>':'')+
      (m.loading?'<div class="loading-table"><span class="spinner"></span>正在读取设备…</div>':
        m.rows.length?'<div class="device-group-list-scroll" role="region" aria-label="分组设备列表" tabindex="0"><table class="device-group-table"><thead><tr>'+checkboxHead+
          '<th>设备名称 / ID</th><th>操作系统</th><th>设备用户</th><th>当前分组</th><th>最近在线</th></tr></thead><tbody>'+tableRows+'</tbody></table></div>':
          '<div class="device-group-empty">'+(m.error?'无法显示设备':'没有匹配的设备')+'</div>')+
      '<div class="device-group-pager"><span>第 '+m.page+' / '+currentPageCount(m)+' 页</span><div class="device-group-page-buttons">'+
      '<button type="button" class="btn btn-sm" data-action="device-group-page" data-page="prev" '+(m.page<=1||m.loading?'disabled':'')+'>上一页</button>'+ 
      '<button type="button" class="btn btn-sm" data-action="device-group-page" data-page="next" '+(m.page>=currentPageCount(m)||m.loading?'disabled':'')+'>下一页</button></div></div>'+
      '<p class="device-group-note">'+(adding?'已在其他分组的设备将转移至当前分组；设备原有的所有者和登记记录保持不变。':'移出分组仅取消设备的分组关系，不删除设备。')+'</p></div>';
    const footer = '<div class="device-group-selection-count">已选择 '+selectedCount+' 台设备</div>'+ 
      '<button type="button" class="btn" data-action="close-modal">关闭</button>'+ 
      (canEdit?'<button type="button" class="btn '+(adding?'btn-primary':'btn-danger')+'" data-action="device-group-submit" '+
      (!selectedCount||m.busy||m.loading?'disabled':'')+'>'+(m.busy?'正在保存…':adding?'加入当前分组':'移出选中设备')+'</button>':'');
    return modal('管理分组设备',name,body,footer).replace('class="drawer"','class="drawer drawer-device-group"');
  }

  async function submit(m) {
    if (!m || m.busy || m.loading || !m.selected.size) return;
    if (m.selected.size > 100) {toast('每次最多选择 100 台设备',true);return;}
    const adding = m.mode === "add";
    const moved = adding ? [...m.selected].filter(id => {
      const details = m.selectedDetails.get(id);
      return details?.group_id && Number(details.group_id) !== Number(m.group.id);
    }).length : 0;
    const description = adding ? (moved ?
      '其中 '+moved+' 台设备当前属于其他分组。确认将选中的 '+m.selected.size+' 台设备加入“'+m.group.name+'”吗？':
      '确认将选中的 '+m.selected.size+' 台设备加入“'+m.group.name+'”吗？'):
      '确认将选中的 '+m.selected.size+' 台设备移出“'+m.group.name+'”吗？设备本身将保留。';
    if (!window.confirm(description)) return;
    m.busy = true;
    render();
    try {
      await request('/device_group/devices/batch','POST',{
        group_id:Number(m.group.id),peer_row_ids:[...m.selected],action:adding?'add':'remove'
      });
      if (manager() !== m) return;
      m.selected.clear();
      m.selectedDetails.clear();
      if (adding) {m.mode = 'members';m.query = '';}
      m.page = 1;
      m.busy = false;
      toast('设备分组已更新');
      await load(m);
      try {
        const result = await request('/device_group/list?'+new URLSearchParams({page:String(state.pageNo),page_size:String(state.pageSize)}));
        if (manager()===m) {state.rows=result.list||[];state.total=Number(result.total)||0;m.group=state.rows.find(x=>x.id===m.group.id)||m.group;render();}
      } catch { /* The member list is already current; a normal page refresh can retry the summary. */ }
    } catch (error) {
      if (manager()!==m) return;
      m.busy = false;
      toast(error.message,true);
      render();
    }
  }

  async function handleAction(action,element) {
    const m = manager();
    if (!m || !action.startsWith('device-group-')) return false;
    if (action === 'device-group-tab') {
      if (m.busy) return true;
      const mode = element.dataset.mode;
      if (!['members','add'].includes(mode) || mode===m.mode) return true;
      if (mode==='add'&&!canPageAction('devicegroups','edit')) return true;
      m.mode = mode;m.page=1;m.query='';m.selected.clear();m.selectedDetails.clear();
      await load(m);
    } else if (action === 'device-group-page') {
      const next = m.page + (element.dataset.page==='next'?1:-1);
      if (next>=1 && next<=currentPageCount(m) && !m.loading && !m.busy) {m.page=next;await load(m);}
    } else if (action === 'device-group-search') {
      if (!m.busy) {clearTimeout(searchTimer);m.page=1;await load(m);}
    } else if (action === 'device-group-select-all') {
      if (m.busy) return true;
      const selectable = m.rows.filter(row => m.mode!=='add'||Number(row.group_id)!==Number(m.group.id));
      const shouldSelect = selectable.some(row => !m.selected.has(Number(row.row_id)));
      if (shouldSelect && new Set([...m.selected,...selectable.map(row=>Number(row.row_id))]).size>100) {
        toast('每次最多选择 100 台设备',true);return true;
      }
      for (const row of selectable) {
        const id=Number(row.row_id);
        if (shouldSelect) {m.selected.add(id);m.selectedDetails.set(id,row);}
        else {m.selected.delete(id);m.selectedDetails.delete(id);}
      }
      render();
    } else if (action === 'device-group-submit') {
      await submit(m);
    }
    return true;
  }

  function handleInput(element) {
    const m = manager();
    if (!m||element.id!=='device-group-query') return false;
    m.query=element.value;
    clearTimeout(searchTimer);
    searchTimer=setTimeout(()=>{if(manager()===m && !m.busy) {m.page=1;load(m);}},320);
    return true;
  }

  function handleChange(element) {
    const m=manager();
    if (!m||!element.classList.contains('device-group-selected')) return false;
    const id=Number(element.value);
    const row=m.rows.find(item=>Number(item.row_id)===id);
    if (!row||!Number.isSafeInteger(id)||id<=0) return true;
    if (element.checked) {
      if (m.selected.size>=100) {element.checked=false;toast('每次最多选择 100 台设备',true);return true;}
      m.selected.add(id);m.selectedDetails.set(id,row);
    } else {m.selected.delete(id);m.selectedDetails.delete(id);}
    render();
    return true;
  }

  return {open,renderModal,handleAction,handleInput,handleChange};
};
