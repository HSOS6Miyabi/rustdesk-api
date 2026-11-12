"use strict";
(() => {
  const root = document.getElementById("app");
  if (!root) return;

  const controlPermissionFields = [
    ["keyboard","键盘和鼠标"],["remote_printer","远程打印机"],
    ["clipboard","剪贴板"],["file","文件传输"],
    ["audio","音频"],["camera","摄像头"],
    ["terminal","远程终端"],["tunnel","TCP 隧道"],
    ["restart","远程重启"],["recording","会话录制"],
    ["block_input","禁止远程设备本地输入"],["remote_modify","修改远程设置"],
    ["privacy_mode","隐私模式"],["usb","USB 设备"]
  ].map(([name,label])=>[name,label,"permission_state"]);
  const permissionFieldNames=new Set(controlPermissionFields.map(field=>field[0]));
  const administratorPermissions = [
    ["users.view","查看用户"],["users.create","创建用户"],["users.delete","删除用户"],
    ["users.edit","编辑用户信息"],["users.edit_password","修改用户密码"],
    ["users.force_logout","结束账户会话"],["users.manage_2fa","管理双重认证"],
    ["devices.view","查看设备"],["devices.create","创建设备"],
    ["devices.edit_info","修改设备信息"],["devices.delete","删除设备"],["devices.transfer","转移设备所有者"],
    ["user_groups.view","查看用户分组"],["user_groups.edit","管理用户分组"],
    ["device_groups.view","查看设备组"],["device_groups.edit","管理设备组"],
    ["audits.view","查看连接和操作记录"],["audits.edit","管理审计记录"],
    ["strategies.view","查看设备策略"],["strategies.edit","管理设备策略"],
    ["control_roles.view","查看控制角色"],["control_roles.edit","管理控制角色"],

    ["address_books.view","查看地址簿"],["address_books.edit","管理地址簿"],
    ["api_tokens.view","查看 API 令牌"],["api_tokens.edit","管理 API 令牌"],
    ["smtp.view","查看 SMTP 设置"],["smtp.edit","修改 SMTP 设置"],
    ["notifications.view","查看邮件通知"],["notifications.edit","管理邮件通知"],
    ["relay_nodes.view","查看中继节点"],["relay_nodes.edit","管理中继节点"],
    ["access_rules.view","查看连接访问规则"],["access_rules.edit","管理连接访问规则"],
    ["device_deployment.view","查看设备部署"],["device_deployment.edit","管理设备部署"],
    ["sessions.view","查看活动连接"],["sessions.edit","结束活动连接"]
  ];
  const pagePrivileges = {
    users:{view:"users.view",create:"users.create",edit:"users.edit",delete:"users.delete"},
    devices:{view:"devices.view",create:"devices.create",edit:"devices.edit_info",delete:"devices.delete"},
    groups:{view:"user_groups.view",create:"user_groups.edit",edit:"user_groups.edit",delete:"user_groups.edit"},
    devicegroups:{view:"device_groups.view",create:"device_groups.edit",edit:"device_groups.edit",delete:"device_groups.edit"},
    connections:{view:"audits.view",edit:"audits.edit"},
    transfers:{view:"audits.view",edit:"audits.edit"},
    alarms:{view:"audits.view",edit:"audits.edit"},
    operations:{view:"audits.view"},
    loginlogs:{view:"audits.view",edit:"audits.edit"},
    policies:{view:"strategies.view",create:"strategies.edit",edit:"strategies.edit",delete:"strategies.edit"},
    policyAssignments:{view:"strategies.view",create:"strategies.edit",edit:"strategies.edit",delete:"strategies.edit"},
    userPolicyAssignments:{view:"strategies.view",create:"strategies.edit",edit:"strategies.edit",delete:"strategies.edit"},
    groupPolicyAssignments:{view:"strategies.view",create:"strategies.edit",edit:"strategies.edit",delete:"strategies.edit"},
    controlRoles:{view:"control_roles.view",create:"control_roles.edit",edit:"control_roles.edit",delete:"control_roles.edit"},
    controlAssignments:{view:"control_roles.view",create:"control_roles.edit",edit:"control_roles.edit",delete:"control_roles.edit"},
    relayNodes:{view:"relay_nodes.view",create:"relay_nodes.edit",edit:"relay_nodes.edit",delete:"relay_nodes.edit"},
    accessRules:{view:"access_rules.view",create:"access_rules.edit",edit:"access_rules.edit",delete:"access_rules.edit"},
    apiTokens:{view:"api_tokens.view",create:"api_tokens.edit",delete:"api_tokens.edit"},
    mail:{view:"smtp.view",edit:"smtp.edit"},
    notifications:{view:"notifications.view",edit:"notifications.edit"},
    deployed:{view:"device_deployment.view",edit:"device_deployment.edit"},
    activeSessions:{view:"sessions.view",edit:"sessions.edit"},
    disconnectHistory:{view:"sessions.view"},
    sessions:{view:"users.force_logout"},
  };
  const pages = {
    overview: {name:"概览",icon:"home",section:"工作空间"},
    devices: {name:"设备管理",icon:"monitor",section:"设备管理",endpoint:"peer",admin:true,search:"keyword",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["os","系统"],["last_online_time","最后在线"],["last_online_ip","最近上线 IP"],["note","设备备注"]],fields:[["id","设备 ID","text",true],["alias","备注"],["hostname","主机名"],["username","操作系统用户"],["os","操作系统"],["group_id","设备组 ID","number"],["note","设备备注","textarea"]]},
    mydevices: {name:"我的设备",icon:"monitor",section:"工作空间",endpoint:"my/peer",noCreate:true,search:"keyword",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["os","系统"],["last_online_time","最后在线"],["last_online_ip","最近上线 IP"],["note","设备备注"]],fields:[["alias","备注"],["note","设备备注","textarea"]]},
    shared: {name:"共享设备",icon:"share",section:"工作空间",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["owner_username","设备所有者"],["sources","共享来源"],["last_online_time","最后在线"]]},
    contacts: {name:"联系人",icon:"users",section:"工作空间"},
    addressbooks: {name:"地址簿",icon:"book",section:"工作空间",endpoint:"address_book",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["username","登录账户"],["platform","平台"]],fields:[["id","设备 ID","text",true],["alias","备注"],["hostname","主机名"],["username","登录账户"],["password","连接密码","password"],["platform","平台"],["user_id","所有者 ID","number"],["collection_id","地址簿 ID","number"],["forceAlwaysRelay","强制使用中继","checkbox"]]},
    collections: {name:"地址簿分类",icon:"folder",section:"工作空间",endpoint:"address_book_collection",columns:[["id","编号"],["name","分类名称"],["user_id","所有者 ID"]],fields:[["name","名称","text",true],["user_id","所有者 ID","number",true]]},
    tags: {name:"设备标签",icon:"tag",section:"工作空间",endpoint:"tag",columns:[["id","编号"],["name","标签名称"],["color","颜色"],["collection_id","地址簿 ID"]],fields:[["name","名称","text",true],["color","颜色","color",true],["user_id","所有者 ID","number"],["collection_id","地址簿 ID","number"]]},
    mygroups: {name:"我的群组",icon:"layers",section:"个人管理",endpoint:"my/group",columns:[["id","编号"],["name","群组名称"],["type","分组类型"]],fields:[["name","名称","text",true],["type","类型","group_type",true]]},
    groups: {name:"用户分组",icon:"layers",section:"成员管理",endpoint:"group",admin:true,columns:[["id","编号"],["name","分组名称"],["type","分组类型"],["owner_user_id","创建者用户 ID"]],fields:[["name","名称","text",true],["type","类型","group_type",true]]},
    users: {name:"用户账户",icon:"users",section:"成员管理",endpoint:"user",admin:true,search:"keyword",columns:[["id","编号"],["username","用户名"],["nickname","昵称"],["email","邮箱"],["group_id","主要分组"],["status","状态"]],fields:[["username","用户名","text",true],["password","初始密码","password",true],["nickname","昵称"],["email","邮箱","email"],["group_id","主要分组 ID","number",true],["is_admin","管理员","checkbox"],["status","账户状态","status"],["remark","备注","textarea"]]},
    deployed: {name:"设备部署记录",icon:"monitor",section:"设备管理",endpoint:"device_deployment",admin:true,readonly:true,columns:[["id","编号"],["device_id","设备 ID"],["owner_user_id","部署账户 ID"],["status","部署状态"],["created_at","部署时间"]]},
    devicegroups: {name:"设备分组",icon:"grid",section:"设备管理",endpoint:"device_group",admin:true,columns:[["id","编号"],["name","名称"]],fields:[["name","名称","text",true]]},
    oauth: {name:"OIDC / SSO",icon:"key",section:"系统管理",endpoint:"oauth",admin:true,columns:[["id","编号"],["op","提供者"],["oauth_type","类型"],["issuer","颁发者"],["auto_group_ids","自动加入分组"]],fields:[["op","提供者标识"],["oauth_type","登录类型","oauth_type",true],["issuer","Issuer URL"],["scopes","Scopes"],["client_id","Client ID","text",true],["client_secret","Client Secret","password",true],["auto_register","允许自动注册","checkbox"],["auto_group_ids","自动加入的用户分组"],["pkce_enable","启用 PKCE","checkbox"],["pkce_method","PKCE 方法"]]},
    trash: {name:"地址簿回收站",icon:"book",section:"个人管理",endpoint:"my/address_book/trash",readonly:true,columns:[["id","设备 ID"],["alias","别名"],["hostname","主机名"],["collection_id","地址簿编号"],["deleted_at","删除时间"]]},
    myRules: {name:"我的地址簿共享规则",icon:"shield",section:"个人管理",endpoint:"my/address_book_collection_rule",columns:[["id","编号"],["collection_id","地址簿 ID"],["type","分享对象"],["to_id","对象 ID"],["rule","权限"]],fields:[["collection_id","地址簿 ID","number",true],["type","对象类型","rule_type",true],["to_id","对象 ID（所有用户填 0）","number",true],["rule","权限级别","rule",true]]},
    rules: {name:"地址簿共享权限",icon:"shield",section:"系统管理",endpoint:"address_book_collection_rule",admin:true,columns:[["id","编号"],["collection_id","地址簿 ID"],["type","对象类型"],["to_id","对象 ID"],["rule","权限级别"]],fields:[["collection_id","地址簿 ID","number",true],["type","对象类型","rule_type",true],["to_id","对象 ID","number",true],["rule","权限级别","rule",true],["user_id","所有者 ID","number"]]},
    server: {name:"服务器管理",icon:"network",section:"服务器管理",admin:true},
    settings: {name:"设置中心",icon:"settings",section:"系统管理",admin:true},
    myconnections: {name:"我的连接记录",icon:"link",section:"我的审计记录",endpoint:"my/audit_conn",readonly:true,columns:[["id","编号"],["direction","方向"],["from_peer","连接方设备 ID"],["from_device_name","连接方设备"],["from_account","连接方账户"],["from_name","设备登录用户名"],["peer_id","被连接设备 ID"],["to_device_name","被连接设备"],["to_account","被连接方账户"],["ip","来源 IP"],["primary_auth","认证方式"],["two_factor","双重认证"],["conn_audit_ref","连接关联标识"],["note","连接备注"],["created_at","时间"]]},
    mytransfers: {name:"我的文件传输",icon:"file",section:"我的审计记录",endpoint:"my/audit_file",readonly:true,columns:[["id","编号"],["direction","方向"],["from_peer","连接方设备 ID"],["from_account","连接方账户"],["peer_id","被连接设备 ID"],["to_account","被连接方账户"],["conn_id","连接编号"],["path","文件路径"],["ip","来源 IP"],["created_at","时间"]]},
    myalarms: {name:"我的安全告警",icon:"shield",section:"我的审计记录",endpoint:"my/audit_alarm",readonly:true,columns:[["id","编号"],["typ","告警类别"],["peer_id","设备 ID"],["conn_id","连接编号"],["info","告警详情"],["created_at","时间"]]},
    alarms: {name:"安全告警",icon:"shield",section:"审计管理",endpoint:"audit_alarm",admin:true,readonly:true,columns:[["id","编号"],["typ","告警类别"],["peer_id","设备 ID"],["from_user_id","发起方用户 ID"],["to_user_id","接收方用户 ID"],["conn_id","连接编号"],["source_ip","来源 IP"],["info","告警详情"],["created_at","时间"]]},
    operations: {name:"管理操作记录",icon:"clock",section:"审计管理",endpoint:"audit_operation",admin:true,readonly:true,columns:[["id","编号"],["account","操作账户"],["method","请求方法"],["path","操作内容"],["client_ip","来源 IP"],["success","结果"],["created_at","时间"]]},
    policies: {name:"设备安全策略",icon:"shield",section:"设备管理",endpoint:"device_policy",admin:true,columns:[["id","编号"],["name","策略名称"],["id_whitelist","设备 ID 白名单"],["ip_whitelist","IP 白名单"],["enable_webrtc","启用 WebRTC"],["trust_duration_days","可信设备有效期（天）"],["status","策略状态"]],fields:[["name","策略名称","text",true],["status","策略状态","status"],["id_whitelist","设备 ID 白名单","textarea"],["ip_whitelist","IP 白名单（单个地址或 CIDR）","textarea"],["enable_webrtc","允许 WebRTC 连接","checkbox"],["trust_duration_days","可信设备有效期（天）","trusted_days"],
      ["access_mode","允许的远程连接方式","policy_access"],
      ["enable_keyboard","允许远程键盘和鼠标","policy_bool"],
      ["enable_clipboard","允许远程剪贴板","policy_bool"],
      ["enable_file_transfer","允许远程文件传输","policy_bool"],
      ["approval_mode","远程连接确认方式","policy_approve"]]},
    policyAssignments: {name:"设备策略分配",icon:"grid",section:"设备管理",endpoint:"device_policy/assignment",admin:true,columns:[["id","编号"],["device_id","设备 ID"],["policy_id","安全策略编号"]],fields:[["device_id","设备 ID","text",true],["policy_id","安全策略编号","number",true]]},
    userPolicyAssignments: {name:"用户策略分配",icon:"users",section:"设备管理",endpoint:"device_policy/user_assignment",admin:true,columns:[["id","编号"],["user_id","用户 ID"],["policy_id","策略编号"]],fields:[["user_id","用户 ID","number",true],["policy_id","策略编号","number",true]]},
    groupPolicyAssignments: {name:"设备组策略分配",icon:"grid",section:"设备管理",endpoint:"device_policy/group_assignment",admin:true,columns:[["id","编号"],["group_id","设备组 ID"],["policy_id","策略编号"]],fields:[["group_id","设备组 ID","number",true],["policy_id","策略编号","number",true]]},
    retention: {name:"日志保留设置",icon:"settings",section:"审计管理",admin:true},
    myloginlogs: {name:"我的登录记录",icon:"clock",section:"我的审计记录",endpoint:"my/login_log",readonly:true,columns:[["id","编号"],["client","客户端"],["ip","IP 地址"],["platform","平台"],["created_at","时间"]]},
    loginlogs: {name:"登录记录",icon:"clock",section:"审计管理",endpoint:"login_log",admin:true,readonly:true,columns:[["id","编号"],["user_id","用户 ID"],["client","客户端"],["ip","IP 地址"],["platform","平台"],["created_at","时间"]]},
    connections: {name:"全部连接记录",icon:"link",section:"审计管理",endpoint:"audit_conn",admin:true,readonly:true,columns:[["id","编号"],["direction","方向"],["from_peer","连接方设备 ID"],["from_device_name","连接方设备"],["from_account","连接方账户"],["from_name","设备登录用户名"],["peer_id","被连接设备 ID"],["to_device_name","被连接设备"],["to_account","被连接方账户"],["ip","来源 IP"],["primary_auth","认证方式"],["two_factor","双重认证"],["conn_audit_ref","连接关联标识"],["note","连接备注"],["created_at","时间"]]},
    transfers: {name:"全部文件传输",icon:"file",section:"审计管理",endpoint:"audit_file",admin:true,readonly:true,columns:[["id","编号"],["direction","方向"],["from_peer","连接方设备 ID"],["from_account","连接方账户"],["peer_id","被连接设备 ID"],["to_account","被连接方账户"],["conn_id","连接编号"],["path","文件路径"],["ip","来源 IP"],["created_at","时间"]]},
    invitations: {name:"成员邮件邀请",icon:"users",section:"成员管理",endpoint:"invite",admin:true,columns:[["id","编号"],["email","收件邮箱"],["group_id","用户组 ID"],["expires_at","有效期截止"],["accepted_at","接受时间"],["revoked_at","撤销时间"]],fields:[["email","成员邮箱","email",true],["group_id","加入的用户组 ID（默认 1）","number"],["expires_in_days","有效天数（默认 7）","number"]]},
    relayNodes: {name:"中继节点管理",icon:"network",section:"系统管理",endpoint:"relay",admin:true,
      columns:[["id","编号"],["name","节点名称"],["address","中继服务器地址"],["subnets","优先使用的网络范围"],["countries","优先使用的国家代码"],["priority","优先级"],["enabled","是否启用"]],
      fields:[["name","节点名称","text",true],["address","中继服务器地址","text",true],
        ["subnets","优先使用的 IP 网络（每行一个 CIDR）","textarea"],["countries","优先使用的国家代码（例如 JP,US）","textarea"],["priority","优先级（越小越高）","number"],
        ["enabled","是否启用","checkbox"]]},
    accessRules: {name:"连接访问规则",icon:"shield",section:"设备管理",endpoint:"access_rule",admin:true,
      columns:[["id","编号"],["name","规则名称"],["source_type","访问方类型"],["source_id","访问方编号"],["target_type","目标类型"],["target_id","目标编号"],["effect","访问结果"],["enabled","是否启用"]],
      fields:[["name","规则名称","text",true],
        ["source_type","允许访问的用户范围","access_source",true],["source_id","来源用户或用户组编号（所有用户填写 0）","number"],
        ["target_type","目标设备范围","access_target",true],["target_id","目标设备 ID 或用户/设备组编号（所有设备留空）"],
        ["effect","匹配时的操作","access_effect",true],["enabled","启用此规则","checkbox"]]},
    adminRoles: {name:"管理员角色",icon:"shield",section:"成员管理",endpoint:"admin_role",admin:true,
      columns:[["id","编号"],["name","角色名称"],["scope_type","管理范围"],["scope_id","范围编号"],["permissions","允许的操作"],["enabled","是否启用"]],
      fields:[["name","角色名称","text",true],["permissions","管理权限","admin_permissions",true],
        ["scope_type","权限适用范围","admin_scope",true],["scope_id","指定用户或群组编号（全局填 0）","number"],
        ["enabled","启用此角色","checkbox"]]},
    adminRoleAssignments: {name:"管理员角色分配",icon:"users",section:"成员管理",endpoint:"admin_role_assignment",admin:true,
      columns:[["id","编号"],["user_id","成员用户 ID"],["role_id","管理员角色 ID"]],
      fields:[["user_id","成员用户 ID","number",true],["role_id","管理员角色 ID","number",true]]},
    controlRoles: {name:"远程控制角色",icon:"shield",section:"设备管理",endpoint:"control_role",admin:true,
      columns:[["id","编号"],["name","角色名称"],["keyboard","键盘和鼠标"],["clipboard","剪贴板"],["file","文件传输"],["terminal","远程终端"],["remote_modify","修改远程设置"]],
      fields:[["name","角色名称","text",true],...controlPermissionFields]},
    controlAssignments: {name:"远程控制角色分配",icon:"grid",section:"设备管理",endpoint:"control_assignment",admin:true,
      columns:[["id","编号"],["target_type","分配范围"],["target_id","设备或账户编号"],["role_id","控制角色编号"]],
      fields:[["target_type","分配范围","control_target",true],["target_id","设备 ID 或用户/设备组编号","text",true],
        ["role_id","控制角色编号","number",true]]},
    mail: {name:"SMTP 邮件服务",icon:"file",section:"系统管理",admin:true},
    clientBuilder: {name:"客户端安装包构建",icon:"package",section:"系统管理",admin:true},
    notifications: {name:"邮件事件通知",icon:"bell",section:"系统管理",endpoint:"notification",admin:true,readonly:true,columns:[["id","编号"],["event_type","事件类型"],["recipient","收件邮箱"],["status","发送状态"],["attempts","尝试次数"],["sent_at","发送时间"],["last_error","最近一次错误"]]},
    apiTokens: {name:"API 管理令牌",icon:"key",section:"系统管理",admin:true,endpoint:"api_token",columns:[["id","编号"],["name","用途"],["prefix","令牌前缀"],["user_id","所属账户 ID"],["permissions","允许的操作"],["expires_at","过期时间"],["revoked_at","撤销时间"],["last_used_at","最近使用"]],fields:[["name","令牌名称","text",true],["user_id","所属账户 ID（留空为自己）","number"],["permissions","允许的操作（以逗号分隔，例如 devices.read）","text"],["expires_in_days","有效天数（0 表示长期有效）","number"]]},
    activeSessions: {name:"活动远程连接",icon:"link",section:"设备管理",admin:true,readonly:true,endpoint:"session",columns:[["device_id","设备 ID"],["conn_id","连接编号"],["last_seen_at","最后一次心跳"]]},
    disconnectHistory: {name:"结束连接记录",icon:"clock",section:"设备管理",admin:true,readonly:true,endpoint:"session/disconnect",columns:[["device_id","设备 ID"],["conn_id","连接编号"],["status","处理状态"],["requested_by","操作账户 ID"],["created_at","请求时间"],["delivered_at","通知时间"],["completed_at","结束时间"]]},
    sessions: {name:"全部登录会话",icon:"lock",section:"系统管理",endpoint:"user_token",admin:true,noCreate:true,columns:[["id","会话编号"],["username","登录账户"],["device_id","设备 ID"],["device_uuid","终端标识"],["login_ip","登录 IP"],["client","客户端"],["platform","系统"],["created_at","登录时间"],["expired_at","有效期截止"],["active","状态"],["current","当前会话"]]},
    mysessions: {name:"我的登录会话",icon:"lock",section:"个人管理",endpoint:"my/user_token",readonly:true,columns:[["id","会话编号"],["device_id","设备 ID"],["device_uuid","终端标识"],["login_ip","登录 IP"],["client","客户端"],["platform","系统"],["created_at","登录时间"],["expired_at","有效期截止"],["active","状态"],["current","当前会话"]]},
    myactiveSessions: {name:"我的活动远程连接",icon:"link",section:"我的审计记录",endpoint:"my/session",readonly:true,columns:[["device_id","设备 ID"],["conn_id","连接编号"],["last_seen_at","最近活动时间"]]},
    myDisconnectHistory: {name:"我的结束连接记录",icon:"clock",section:"我的审计记录",endpoint:"my/session/disconnect",readonly:true,columns:[["device_id","设备 ID"],["conn_id","连接编号"],["status","处理状态"],["created_at","请求时间"],["delivered_at","通知时间"],["completed_at","结束时间"]]},
    push: {name:"消息推送",icon:"bell",section:"账户"},
    profile: {name:"个人设置",icon:"settings",section:"账户"}
  };
  const drawings = {
    home:'<path d="M3 10 12 3l9 7v11h-6v-7H9v7H3z"/>',
    monitor:'<rect x="3" y="4" width="18" height="14" rx="2"/><path d="M8 22h8m-4-4v4"/>',
    users:'<circle cx="9" cy="7" r="4"/><path d="M2 21v-2a5 5 0 0 1 5-5h4a5 5 0 0 1 5 5v2m0-16a4 4 0 0 1 0 8m6 8v-2a5 5 0 0 0-3-4.5"/>',
    share:'<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="m8.7 13.4 6.6 4.2M15.3 6.5l-6.6 4.1"/>',
    book:'<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>',
    folder:'<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
    tag:'<path d="M20.6 13.4 13.4 20.6a2 2 0 0 1-2.8 0L2 12V2h10l8.6 8.6a2 2 0 0 1 0 2.8z"/><circle cx="7" cy="7" r="1"/>',
    layers:'<path d="m12 2 10 5-10 5L2 7zm-10 10 10 5 10-5M2 17l10 5 10-5"/>',
    grid:'<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>',
    shield:'<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="m9 12 2 2 4-4"/>',
    key:'<circle cx="7.5" cy="15.5" r="5.5"/><path d="m21 2-9.6 9.6M16 4l4 4m-2 2 3 3"/>',
    clock:'<circle cx="12" cy="12" r="9"/><path d="M12 6v6l4 2"/>',
    link:'<path d="M10 13a5 5 0 0 0 7.1 0l3-3A5 5 0 0 0 13 2.9l-1.7 1.7M14 11a5 5 0 0 0-7.1 0l-3 3A5 5 0 0 0 11 21.1l1.7-1.7"/>',
    file:'<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M8 13h8m-8 4h8"/>',
    lock:'<rect x="4" y="11" width="16" height="11" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/>',
    terminal:'<rect x="2" y="3" width="20" height="18" rx="2"/><path d="m6 8 4 4-4 4m7 4h5"/>',
    settings:'<circle cx="12" cy="12" r="3"/><path d="M20 12a8 8 0 0 0-.2-1.9l2-1.5-2-3.4-2.4 1a9 9 0 0 0-3.4-2V2h-4v2.2a9 9 0 0 0-3.4 2l-2.4-1-2 3.4 2 1.5a8 8 0 0 0 0 3.8l-2 1.5 2 3.4 2.4-1a9 9 0 0 0 3.4 2V22h4v-2.2a9 9 0 0 0 3.4-2l2.4 1 2-3.4-2-1.5A8 8 0 0 0 20 12z"/>',
    search:'<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>',
    plus:'<path d="M12 5v14M5 12h14"/>',
    refresh:'<path d="M20 7v5h-5M4 17v-5h5"/><path d="M5 9a8 8 0 0 1 13-3l2 3M4 15l2 3a8 8 0 0 0 13-3"/>',
    menu:'<path d="M4 6h16M4 12h16M4 18h16"/>',
    close:'<path d="M5 5l14 14M19 5 5 19"/>',
    sun:'<circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5M18 18l1.5 1.5M5 19l1.5-1.5M18 6l1.5-1.5"/>',
    moon:'<path d="M20 15.4A8 8 0 0 1 8.6 4a8.5 8.5 0 1 0 11.4 11.4z"/>',
    logout:'<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5m5 5H9"/>',
    arrow:'<path d="M5 12h14m-7-7 7 7-7 7"/>',
    info:'<circle cx="12" cy="12" r="10"/><path d="M12 11v6m0-10h.01"/>',
    chevron:'<path d="m6 9 6 6 6-6"/>'
  };
  function icon(name,size=20) { return '<svg width="'+size+'" height="'+size+'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'+(drawings[name]||drawings.info)+'</svg>'; }
  function esc(v) {return String(v===undefined||v===null?"":v).replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));}
  function txt(v) {return v===null||v===undefined||v===""?"—":Array.isArray(v)?v.join(", "):typeof v==="object"?JSON.stringify(v):String(v);}
  function date(v) {if(!v)return "—";if(typeof v==="number"||/^[0-9]{10,13}$/.test(String(v))){const n=Number(v);return new Date(n<1e12?n*1000:n).toLocaleString("zh-CN",{hour12:false});}return esc(txt(v));}
  const geoIPFields=new Set(["ip","login_ip","client_ip","source_ip","last_online_ip","remote_ip","public_ip","private_ip","device_ip"]);
  const deviceReferenceFields=new Set(["device_id","peer_id","from_peer","to_peer"]);
  const deviceRowPages=new Set(["devices","mydevices","shared","addressbooks","trash"]);
  function deviceStatusDot(id){
    const key=String(id??"").trim();
    if(!key)return "";
    const active=state.devicePresence[key];
    const appearance=active===true?"online":active===false?"offline":"unknown";
    const label=active===true?"在线":active===false?"离线":"状态暂不可用";
    return '<span class="device-status-dot '+appearance+'" role="img" aria-label="'+label+'" title="'+label+'"></span>';
  }
  function collectDeviceIDs(raw,out,deviceRows=false){
    if(Array.isArray(raw)){
      for(const item of raw)collectDeviceIDs(item,out,deviceRows);
      return;
    }
    if(!raw||typeof raw!=="object")return;
    for(const [key,value] of Object.entries(raw)){
      if(typeof value==="string"&&(deviceReferenceFields.has(key)||(deviceRows&&key==="id"))){
        const deviceID=value.trim();
        if(deviceID&&deviceID.length<=253)out.add(deviceID);
      }else if(key==="id_whitelist"&&typeof value==="string"){
        for(const id of value.split(/[,;\s]+/).filter(Boolean).slice(0,256))if(id.length<=253)out.add(id);
      }else if(key==="target_id"&&raw.target_type==="device"&&typeof value==="string"&&value)out.add(value);
      else if(value&&typeof value==="object"){
        collectDeviceIDs(value,out,deviceRows);
      }
    }
  }
  function findDisplayIP(value){
    const raw=String(value||"").trim();
    if(/^(?:\d{1,3}\.){3}\d{1,3}$/.test(raw)||/^[a-fA-F0-9:]+:[a-fA-F0-9:]+$/.test(raw))return raw;
    const bracket=raw.match(/\[([a-fA-F0-9:]+)\]/);
    if(bracket)return bracket[1];
    const ipv4=raw.match(/(?:^|[/:])((?:\d{1,3}\.){3}\d{1,3})(?::\d+)?(?:\/|$)/);
    return ipv4?ipv4[1]:"";
  }
  function geoIPText(ip){
    const name=String(state.geoLocations[ip]||"位置未知");
    return '<span class="ip-location">（'+esc(name)+'）</span>';
  }
  function showIP(value){
    const raw=String(value||"").trim();
    if(!raw)return "—";
    if(raw.includes("/")&&raw.match(/\/\d{1,3}$/))return esc(raw)+'<span class="ip-location">（IP 地址段）</span>';
    const ip=findDisplayIP(raw);
    if(!ip)return esc(raw);
    return esc(raw)+geoIPText(ip);
  }
  function showIPList(value){
    const raw=String(value||"").trim();
    if(!raw)return "—";
    return raw.split(/[,;\n\r]+/).map(part=>showIP(part)).join("<span>, </span>");
  }
  function showServerAddress(value){
    const raw=String(value||"").trim();
    const ip=findDisplayIP(raw);
    return esc(txt(value))+(ip?geoIPText(ip):"");
  }
  function collectGeoIP(raw,out){
    if(Array.isArray(raw)){for(const item of raw)collectGeoIP(item,out);return;}
    if(!raw||typeof raw!=="object")return;
    for(const [key,value] of Object.entries(raw)){
      if(geoIPFields.has(key) && typeof value==="string"){
        const ip=findDisplayIP(value);
        if(ip)out.add(ip);
      }else if(key==="ip_whitelist"&&typeof value==="string"){
        for(const part of value.split(/[,;\n\r]+/)){
          if(!part.includes("/")){
            const ip=findDisplayIP(part);
            if(ip)out.add(ip);
          }
        }
      }else if(["id_server","relay_server","api_server","host"].includes(key)&&typeof value==="string"){
        const ip=findDisplayIP(value);
        if(ip)out.add(ip);
      }else if(value&&typeof value==="object"){
        collectGeoIP(value,out);
      }
    }
  }
  async function loadGeoIPLocations(){
    const candidates=new Set(),devices=new Set();
    const hasRows=state.page==="shared"||Boolean(pages[activeKey()]?.endpoint);
    if(hasRows){
      collectGeoIP(state.rows,candidates);
      collectDeviceIDs(state.rows,devices,deviceRowPages.has(state.page));
    }
    if(state.page==="overview"){
      collectGeoIP(state.metrics?.recent,candidates);
      collectDeviceIDs(state.metrics?.recent,devices,true);
    }
    if(state.page==="server")collectGeoIP(state.server,candidates);
    const ipList=[...candidates].filter(ip=>!(ip in state.geoLocations)).slice(0,256);
    const deviceIDs=[...devices].slice(0,256);
    if(!ipList.length&&!deviceIDs.length)return;
    for(const id of deviceIDs)state.devicePresence[id]=null;
    try{
      const result=await request("/geoip/lookup","POST",{ips:ipList,device_ids:deviceIDs});
      Object.assign(state.geoLocations,result.locations||{});
      const statuses=result.device_status||{};
      for(const id of deviceIDs)state.devicePresence[id]=Object.prototype.hasOwnProperty.call(statuses,id)?statuses[id]===true:null;
    }catch{
      for(const ip of ipList)state.geoLocations[ip]="位置未知";
    }
  }
  const userReferenceFields=new Set(["user_id","owner_user_id","from_user_id","to_user_id","requested_by"]);
  const accountReferenceFields=new Set(["owner_username","from_account","to_account","account"]);
  function entityLink(kind,id,text,showStatus=false){
    if(id===undefined||id===null||String(id).trim()==="")return esc(txt(text));
    const label=String(text??id);
    return '<button type="button" class="entity-link" data-action="entity-detail" data-entity-type="'+esc(kind)+
      '" data-entity-id="'+esc(id)+'" data-entity-label="'+esc(label)+'" title="查看'+(kind==="device"?"设备":"用户")+'信息">'+
      (showStatus?deviceStatusDot(id):"")+'<span>'+esc(label)+'</span></button>';
  }
  function linkedDeviceID(row,k,v){
    if(deviceReferenceFields.has(k))return String(v??"");
    if(k==="id"&&deviceRowPages.has(state.page)&&typeof v==="string")return v;
    if(["alias","hostname"].includes(k)&&deviceRowPages.has(state.page))return String(row?.id??"");
    if(k==="from_device_name")return String(row?.from_peer||"");
    if(k==="to_device_name")return String(row?.peer_id||"");
    if(k==="target_id"&&row?.target_type==="device")return String(v??"");
    return "";
  }
  function linkedAccountID(row,k,v){
    if(userReferenceFields.has(k))return String(v??"");
    if(k==="id"&&state.page==="users")return String(v??"");
    if(["username","nickname"].includes(k)&&state.page==="users")return String(row?.id||v||"");
    if(k==="owner_username")return String(row?.owner_user_id||row?.user_id||v||"");
    if(k==="from_account")return String(row?.from_user_id||v||"");
    if(k==="to_account")return String(row?.to_user_id||v||"");
    if(k==="account")return String(v||"");
    if(["source_id","target_id"].includes(k)&&
      (k==="source_id"?row?.source_type==="user":row?.target_type==="user"))return String(v??"");
    if(k==="to_id"&&Number(row?.type)===1)return String(v??"");
    return "";
  }

  const referenceSources={
    user_group:{path:"/group/list",name:"name"},
    device_group:{path:"/device_group/list",name:"name"},
    user:{path:"/user/list",name:"nickname",fallback:"username"},
    collection:{path:"/address_book_collection/list",name:"name"},
    policy:{path:"/device_policy/list",name:"name"},
    control_role:{path:"/control_role/list",name:"name"},
    admin_role:{path:"/admin_role/list",name:"name"}
  };
  function referenceKind(key,row={}){
    if(key==="group_id"||key==="auto_group_ids")return ["devices","groupPolicyAssignments"].includes(state.page)?"device_group":"user_group";
    if(["user_id","owner_user_id","from_user_id","to_user_id","requested_by"].includes(key))return "user";
    if(key==="collection_id")return "collection";
    if(key==="policy_id")return "policy";
    if(key==="role_id")return state.page==="adminRoleAssignments"?"admin_role":"control_role";
    if(key==="to_id")return Number(row.type)===2?"user_group":Number(row.type)===1?"user":"";
    if(key==="source_id")return row.source_type==="group"?"user_group":row.source_type==="user"?"user":"";
    if(key==="target_id")return row.target_type==="device_group"?"device_group":row.target_type==="user"?"user":"";
    if(key==="scope_id")return row.scope_type==="group"?"user_group":row.scope_type==="individual"?"user":"";
    return "";
  }
  function referenceText(kind,id){
    if(id===undefined||id===null||String(id)===""||String(id)==="0")return "未设置";
    const name=state.referenceNames[kind]?.[String(id)];
    return name?name+"（编号 "+id+"）":"编号 "+id;
  }
  async function loadPageReferences(page){
    const fields=[...(page?.columns||[]).map(c=>c[0]),...(page?.fields||[]).map(c=>c[0])];
    const examples=[{},...state.rows.slice(0,200)];
    const kinds=[...new Set(fields.flatMap(key=>examples.map(row=>referenceKind(key,row))).filter(Boolean))];
    await Promise.all(kinds.filter(kind=>!state.referenceLoaded[kind]).map(async kind=>{
      const source=referenceSources[kind];if(!source)return;
      const path=kind==="user_group"&&!state.isAdmin?"/my/group/list":
        kind==="collection"&&!state.isAdmin?"/my/address_book_collection/list":source.path;
      state.referenceLoaded[kind]=true;
      try{
        const data=await request(path+"?page=1&page_size=200");
        const rows=Array.isArray(data)?data:data?.list||[];
        state.referenceRows[kind]=rows;
        state.referenceNames[kind]=Object.fromEntries(rows.map(row=>[
          String(row.id),String(row[source.name]||row[source.fallback]||row.username||row.id)
        ]));
      }catch{
        state.referenceRows[kind]=[];
        state.referenceNames[kind]={};
      }
    }));
  }
  function colValue(v,k,row){
    if(k==="to_id"&&Number(row?.type)===3)return "所有用户";
    if(k==="source_id"&&row?.source_type==="all")return "所有已登录用户";
    if(k==="target_id"&&row?.target_type==="all")return "所有设备";
    if(k==="scope_id"&&row?.scope_type==="global")return "全部账户";
    if(k==="auto_group_ids"){
      const ids=Array.isArray(v)?v:String(v||"").split(/[,;，\s]+/).filter(Boolean);
      return ids.length?ids.map(id=>'<span class="reference-label">'+esc(referenceText("user_group",id))+'</span>').join("、"):"—";
    }
    const refKind=referenceKind(k,row);
    if(refKind&&v!==undefined&&v!==null&&String(v)!==""){
      const name=referenceText(refKind,v);
      if(refKind==="user"&&String(v)!=="0")return entityLink("user",v,name);
      return '<span class="reference-label">'+esc(name)+'</span>';
    }
    if(k==="id_whitelist"&&typeof v==="string"&&v.trim()){
      return v.split(/[,;\s]+/).filter(Boolean).map(id=>entityLink("device",id,id,true)).join('<span class="entity-separator">, </span>');
    }
    const device=linkedDeviceID(row,k,v);
    if(device&&v)return entityLink("device",device,v,true);
    const account=linkedAccountID(row,k,v);
    if(account&&v!==undefined&&v!==null&&String(v)!=="")return entityLink("user",account,v,false);
    if(geoIPFields.has(k))return showIP(v);
    if(k==="ip_whitelist")return showIPList(v);
    if(k==="sources"&&Array.isArray(v))return v.map(x=>'<span class="pill '+(x.type==="contact"?"good":"")+'" style="margin:2px">'+esc(x.type==="contact"?"联系人："+x.name:"群组："+x.name)+'</span>').join("");
    if(k==="type"&&(v===1||v===2||v===3))return ["","普通组","共享组","公共锚点组"][v];
    if(permissionFieldNames.has(k))return ["沿用设备设置","禁止","允许"][Number(v)]||"—";
    if(k==="target_type")return ({device:"单台设备",user:"设备所有者",device_group:"设备组"}[v]||v);
    if(k==="scope_type")return ({global:"全部账户",group:"指定用户组",individual:"指定用户"}[v]||v);
    if(k==="source_type")return ({all:"所有已登录用户",user:"指定用户",group:"指定用户组"}[v]||v);
    if(k==="target_type")return ({all:"全部设备",device:"指定设备",user:"指定所有者",device_group:"指定设备组"}[v]||v);
    if(k==="effect")return ({allow:"允许访问",deny:"禁止访问"}[v]||v);
    if(k==="direction")return v==="outgoing"?"我发起的连接":v==="incoming"?"连接我的设备":v==="both"?"双方均为我的设备":"—";
    if(k==="primary_auth")return (["未报告","点击授权","临时密码","永久密码","切换控制方向"][Number(v)]||"其他方式")+" ("+esc(txt(v))+")";
    if(k==="two_factor")return (["未报告","TOTP 动态验证码","可信设备"][Number(v)]||"其他方式")+" ("+esc(txt(v))+")";
    if(k==="typ")return ({0:"IP 白名单拒绝",1:"密码尝试次数过多",2:"短时间连续登录失败",6:"IPv6 地址段登录尝试过多",7:"系统账户登录暂缓",8:"系统账户登录并发限制",9:"会话权限违规",10:"设备 ID 白名单拒绝"}[Number(v)]||"其他告警")+" ("+esc(txt(v))+")";
    if(k==="success")return v?'<span class="pill good">成功</span>':'<span class="pill warn">失败</span>';
    if(k==="is_admin")return v?"管理员":"普通用户";
    if(k==="enable_webrtc")return v?"启用":"停用";
    if(k==="enabled")return v?"已启用":"已停用";
    if(k==="event_type")return esc({
      "login.failure":"登录失败","deployment.changed":"设备登记变更",
      "session.disconnect_requested":"远程连接结束请求",
      "session.ended":"远程连接已结束","audit.file":"文件传输"
    }[v]||v);
    if(k==="color"&&v!==undefined&&v!==null){
      const argb=Number(v)>>>0;
      const rgb="#"+(argb&0xffffff).toString(16).padStart(6,"0");
      const alpha=(argb>>>24).toString(16).padStart(2,"0");
      return '<span class="color-label"><span class="color-swatch" style="background:'+rgb+alpha+'"></span>'+esc("#"+argb.toString(16).padStart(8,"0").toUpperCase())+'</span>';
    }
    if(k==="active")return v?'<span class="pill good">有效</span>':'<span class="pill muted">已过期</span>';
    if(k==="current")return v?'<span class="pill good">当前登录</span>':"—";
    if(k==="login_ip")return v?showIP(v):"未记录";
    if(k==="device_uuid"&&!v)return "未提供终端标识";
    if(k==="status"&&typeof v==="string")return esc({
      approved:"已登记",revoked:"已撤销",pending:"等待处理",
      delivered:"已通知客户端",completed:"连接已结束",expired:"已过期",
      sending:"正在发送",retry:"等待重试",sent:"发送成功",failed:"发送失败"
    }[v]||v);
    if(k==="status")return '<span class="pill '+(Number(v)===1?"good":"warn")+'">'+(Number(v)===1?"启用":"停用")+"</span>";
    if(["sent_at","created_at","last_online_time","expire","expired_at","expires_at","revoked_at","last_used_at","last_seen_at","delivered_at","completed_at","deleted_at","accepted_at"].includes(k))return date(v);
    if(typeof v==="boolean")return v?"是":"否";
    return esc(txt(v));
  }
  const state={
    token:sessionStorage.getItem("rustdesk.console.token")||"",user:null,isAdmin:false,
    page:"overview",bookTab:"entries",pageNo:1,pageSize:20,rows:[],total:0,loading:false,error:"",
    sort:{},navScroll:0,navCollapsed:JSON.parse(localStorage.getItem("rustdesk.console.nav-collapsed")||"{}"),
    query:"",sourceType:"all",sourceId:"0",contacts:{outgoing:[],incoming:[]},
    groups:[],referenceNames:{},referenceLoaded:{},referenceRows:{},showAllColumns:{},server:{},serverTab:"overview",serverResults:{},serverError:"",commands:[],commandSelected:"id-relay-read",commandOutput:"",pickerDialog:null,modal:null,mobile:false,metrics:null,settings:null,settingsOverview:null,settingsHistory:[],settingsFailed:[],settingsFailedCategory:"push",settingsSearch:"",settingsCategory:"all",retention:null,cleanupPreview:null,cleanupRequest:{kind:"all",mode:"retention",days:30},mail:null,notifications:null,pushChannels:[],pushEvents:[],pushHistory:[],pushTotal:0,pushEditing:null,geoLocations:{},devicePresence:{},clientProfiles:[],clientTasks:[],clientEditing:null,mfa:null,emailVerification:null,passkeys:[],passkeysError:"",
    options:{ops:[],need_captcha:false},captcha:null,config:{},
    theme:localStorage.getItem("rustdesk.console.theme")==="dark"?"dark":"light"
  };
  document.documentElement.dataset.theme=state.theme;
  const API="/api/admin";
  let seq=0,debounce=0;
  async function request(route,method="GET",body) {
    const ctl=new AbortController(), timer=setTimeout(()=>ctl.abort(),18000);
    const headers={"Accept":"application/json","Accept-Language":"zh-CN"};
    if(state.token)headers["api-token"]=state.token;
    if(body!==undefined)headers["Content-Type"]="application/json";
    try{
      const res=await fetch(API+route,{method,headers,body:body===undefined?undefined:JSON.stringify(body),signal:ctl.signal,credentials:"same-origin"});
      let data={};try{data=await res.json();}catch{throw Error("服务器返回的数据无法读取");}
      if(!res.ok)throw Error(data.error||data.message||"HTTP "+res.status);
      if(typeof data.code==="number"&&data.code!==0)throw Error(data.message||"操作失败");
      return data.data===undefined?data:data.data;
    }catch(e){if(e.name==="AbortError")throw Error("请求超时");throw e;}finally{clearTimeout(timer);}
  }
  function toast(message,error=false){
    let wrap=document.querySelector(".toast-wrap");
    if(!wrap){wrap=document.createElement("div");wrap.className="toast-wrap";document.body.append(wrap);}
    const node=document.createElement("div");node.className="toast"+(error?" error":"");node.role="status";node.textContent=String(message);
    wrap.append(node);setTimeout(()=>node.remove(),4300);
  }
  function hasPrivilege(permission){
    if(state.isAdmin)return true;
    return !!permission&&Array.isArray(state.user?.route_names)&&state.user.route_names.includes(permission);
  }
  function canPageAction(k,action){
    if(state.isAdmin)return true;
    return hasPrivilege(pagePrivileges[k]?.[action]);
  }
  function available(k){
    const page=pages[k];
    return !!page&&(!page.admin||state.isAdmin||canPageAction(k,"view"));
  }
  function activeKey(){return state.page==="addressbooks"&&state.bookTab==="collections"?"collections":state.page;}
  function endpoint(k) {
    const p=pages[k];
    const delegatedDeviceView=k==="devices"&&hasPrivilege("devices.view");
    return !state.isAdmin&&!delegatedDeviceView&&["peer","address_book","address_book_collection","tag","login_log"].includes(p.endpoint)?"/my/"+p.endpoint:"/"+p.endpoint;
  }
  function pageHeading(title,subtitle,actions=""){
    return '<div class="page-head"><div><p class="eyebrow">RUSTDESK / CONTROL CENTER</p><h1>'+esc(title)+'</h1><p class="subtitle">'+esc(subtitle)+'</p></div><div class="head-actions">'+actions+'</div></div>';
  }
  function button(label,action,primary=false){return '<button type="button" class="btn '+(primary?"btn-primary":"")+'" data-action="'+esc(action)+'">'+esc(label)+'</button>';}
  function brand(){return '<span class="brand-symbol" aria-hidden="true"><span></span><span></span></span><span class="brand-copy"><span class="side-name">RustDesk</span><span class="side-sub">CONTROL CENTER</span></span>';}
  function login(){
    const options=state.options||{};
    const cap=options.need_captcha?'<div class="login-field"><label for="captcha">验证码</label><div style="display:flex;gap:10px"><input name="captcha" id="captcha" required placeholder="输入验证码"><button class="btn" type="button" data-action="captcha">刷新</button></div><input type="hidden" name="captcha_id" value="'+esc(state.captcha&&state.captcha.id||"")+'">'+(state.captcha&&state.captcha.b64?'<img alt="验证码" style="max-height:72px;max-width:220px;margin-top:10px" src="'+esc(state.captcha.b64.startsWith("data:")?state.captcha.b64:"data:image/png;base64,"+state.captcha.b64)+'">':"")+"</div>":"";
    const passwordForm=options.disable_pwd?'<p class="warning">当前服务已关闭密码登录，请选择下方的单点登录服务。</p>':'<form id="login-form"><div class="login-field"><label for="username">用户名</label><input id="username" autocomplete="username" name="username" placeholder="输入用户名" required></div><div class="login-field"><label for="password">密码</label><input type="password" id="password" name="password" autocomplete="current-password" placeholder="输入密码" required></div>'+cap+'<button type="submit" class="btn btn-primary login-submit">登录控制台 '+icon("arrow",17)+'</button></form><button type="button" class="btn" data-action="forgot-password">忘记密码</button>';
    const ops=(options.ops||[]).map(op=>{const n=typeof op==="string"?op:op.op||op.name;return n?'<button type="button" class="btn" data-action="oidc" data-id="'+esc(n)+'">使用 '+esc(n)+' 登录 '+icon("arrow",14)+'</button>':"";}).join("");
    const passkeyEntry='<button type="button" class="btn passkey-login-button" data-action="passkey-login">'+icon("lock",17)+' 使用 Passkey 登录</button>';
    root.innerHTML='<div class="login-shell"><section class="login-art"><div class="brand-row">'+brand()+'</div><div><p class="eyebrow" style="color:#83e7d4">PRIVATE INFRASTRUCTURE</p><h1>远程设备管理，<br><em>一切尽在掌握。</em></h1><p>统一管理远程设备、联系人、共享群组与用户账户。界面可适应电脑、平板和手机。</p></div><div class="login-foot">RustDesk Self-hosted Control Center</div></section><section class="login-right"><div class="login-card"><p class="eyebrow">WELCOME BACK</p><h2>登录控制台</h2><p class="login-intro">'+esc(state.config.title||"请使用 RustDesk API 账户登录")+'</p>'+passwordForm+'<div class="login-options">'+passkeyEntry+ops+'</div><p class="login-help">首次安装的管理员密码可在服务端启动日志中查看。请通过 HTTPS 连接管理页面。</p></div></section></div>';
    root.innerHTML+=loginRecoveryModal();
  }
  function loginRecoveryModal(){
    const current=state.modal;
    if(!current)return "";
    if(current.kind==="mfa-login"){
      const emailMethod=current.method==="email_check";
      const label=emailMethod?"邮箱验证码":"动态验证码或一次性恢复码";
      const tip=emailMethod?"请输入发送到该账户邮箱的 6 位验证码。":
        "使用身份验证器中的 6 位验证码，或之前保存的恢复码。";
      const body='<form id="mfa-login-form"><div class="field"><label for="mfa-login-code">'+label+'</label>'+
        '<input id="mfa-login-code" name="code" autocomplete="one-time-code" minlength="6" maxlength="20" required autofocus>'+
        '<small>'+tip+'</small></div></form>';
      return modal("双重认证","账户："+current.username,body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="mfa-login-form">完成登录</button>');
    }
    if(current.kind==="forgot-password"){
      const body='<form id="forgot-password-form"><div class="field"><label for="recovery-email">账户邮箱</label>'+
        '<input type="email" name="email" id="recovery-email" autocomplete="email" required></div></form>';
      return modal("找回密码","将向已登记的账户邮箱发送密码恢复链接。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" type="submit" form="forgot-password-form">发送恢复邮件</button>');
    }
    if(current.kind==="accept-invitation"){
      const body='<form id="accept-invitation-form"><div class="form-grid">'+
        '<div class="field"><label for="invited-username">用户名</label>'+
        '<input id="invited-username" name="username" autocomplete="username" minlength="3" maxlength="64" required></div>'+
        '<div class="field"><label for="invited-nickname">显示名称</label>'+
        '<input id="invited-nickname" name="nickname" autocomplete="name"></div>'+
        '<div class="field"><label for="invited-password">登录密码</label>'+
        '<input id="invited-password" name="password" type="password" autocomplete="new-password" minlength="8" maxlength="72" required></div>'+
        '<div class="field"><label for="invited-password-confirm">确认密码</label>'+
        '<input id="invited-password-confirm" name="confirm_password" type="password" autocomplete="new-password" minlength="8" maxlength="72" required></div>'+
        '</div></form>';
      return modal("接受团队邀请","创建账号后，即可使用 RustDesk 自托管服务。",body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="accept-invitation-form">创建账户</button>');
    }
    if(current.kind==="public-reset-password"){
      const body='<form id="public-password-reset-form"><div class="form-grid">'+
        '<div class="field"><label for="recovery-new-password">新密码</label>'+
        '<input type="password" name="password" id="recovery-new-password" minlength="8" maxlength="72" autocomplete="new-password" required></div>'+
        '<div class="field"><label for="recovery-confirm-password">确认新密码</label>'+
        '<input type="password" name="confirm_password" id="recovery-confirm-password" minlength="8" maxlength="72" autocomplete="new-password" required></div>'+
        '</div></form>';
      return modal("设置新密码","使用邮件中的一次性恢复链接。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" type="submit" form="public-password-reset-form">保存新密码</button>');
    }
    return "";
  }
  function shell(){
    const username=state.user&&state.user.nickname||state.user&&state.user.username||"用户";
    const sections=["工作空间","个人管理","我的审计记录","成员管理","设备管理","审计管理","服务器管理","系统管理","账户","客户端"];
    const items=Object.entries(pages).filter(([key])=>key!=="collections"&&available(key));
    let nav="";
    for(const section of sections){
      const links=items.filter(([,p])=>p.section===section);
      if(section==="客户端")links.push(["webclient",{name:"WebClient v2",icon:"monitor"}]);
      if(!links.length)continue;
      const collapsed=!!state.navCollapsed[section];
      nav+='<button type="button" class="side-section-toggle" data-action="toggle-section" data-section="'+esc(section)+'" aria-expanded="'+(!collapsed)+'">'+
        '<span>'+esc(section)+'</span>'+icon("chevron",16)+'</button><div class="side-section-links" '+(collapsed?"hidden":"")+'>';
      for(const [key,p] of links){
        nav+='<button class="nav-link '+(key===state.page?"active":"")+'" type="button" '+
          (key==="webclient"?'data-action="webclient"':'data-view="'+esc(key)+'"')+
          ' '+(key===state.page?'aria-current="page"':'')+'>'+icon(p.icon,18)+esc(p.name)+'</button>';
      }
      nav+="</div>";
    }
    return '<div class="workspace"><div class="sidebar-backdrop '+(state.mobile?"show":"")+'" data-action="mobile-close"></div><aside class="sidebar '+(state.mobile?"open":"")+'"><div class="side-head">'+brand()+'</div><nav class="side-scroll" aria-label="导航">'+nav+'</nav><div class="side-bottom"><div class="side-avatar"><span class="avatar">'+esc(username[0]||"R").toUpperCase()+'</span><div class="side-account"><strong>'+esc(username)+'</strong><small>'+(state.isAdmin?"系统管理员":"普通用户")+'</small></div></div></div></aside><div class="main"><header class="topbar"><div class="breadcrumbs"><button class="icon-btn mobile-only" data-action="mobile-open" aria-label="打开导航">'+icon("menu")+'</button><span>'+esc(pages[state.page].section)+'</span><span>/</span><span class="crumb-current">'+esc(pages[state.page].name)+'</span></div><div class="top-actions"><button class="icon-btn" data-action="refresh" title="刷新">'+icon("refresh")+'</button><button class="icon-btn" data-action="theme" title="切换主题">'+icon(state.theme==="dark"?"sun":"moon")+'</button><button class="top-user" data-view="profile"><span class="mini-avatar">'+esc(username[0]||"R").toUpperCase()+'</span>'+esc(username)+'</button><button class="icon-btn" data-action="logout" title="退出登录">'+icon("logout")+'</button></div></header><main class="content">'+renderPage()+'</main></div></div>'+renderModal();
  }
  function render(){
    if(!state.token||!state.user){login();return;}
    const oldScroll=root.querySelector(".side-scroll");
    if(oldScroll)state.navScroll=oldScroll.scrollTop;
    root.innerHTML=shell();
    const newScroll=root.querySelector(".side-scroll");
    if(newScroll)newScroll.scrollTop=state.navScroll;
  }
  function view(k,query=""){
    if(k==="devices"&&!available("devices"))k="mydevices";
    if(!available(k))return;
    state.page=k;if(k==="addressbooks")state.bookTab="entries";state.pageNo=1;state.query=query;state.sourceType="all";state.sourceId="0";state.mobile=false;state.modal=null;state.error="";
    location.hash="#/"+k+(query?"?search="+encodeURIComponent(query):"");render();load();
  }
  function returnToWebClient(){
    if(sessionStorage.getItem("rustdesk.console.return")!=="/webclient2/")return false;
    sessionStorage.removeItem("rustdesk.console.return");
    window.location.assign("/webclient2/");
    return true;
  }
  async function bootstrap(){
    const recoveryHash=location.hash||"";
    if(recoveryHash.startsWith("#/invite?")){
      const token=new URLSearchParams(recoveryHash.slice(recoveryHash.indexOf("?")+1)).get("token");
      if(token){
        state.token="";state.user=null;
        sessionStorage.removeItem("rustdesk.console.token");
        state.modal={kind:"accept-invitation",token};
      }
    }
    if(recoveryHash.startsWith("#/password-reset?")){
      const token=new URLSearchParams(recoveryHash.slice(recoveryHash.indexOf("?")+1)).get("token");
      if(token){
        state.token="";state.user=null;
        sessionStorage.removeItem("rustdesk.console.token");
        state.modal={kind:"public-reset-password",token};
      }
    }
    try{state.config=await request("/config/admin");}catch{state.config={};}
    if(!state.token){
      const pending=sessionStorage.getItem("rustdesk.console.oidc");
      if(pending){try{const data=await request("/oidc/auth-query?code="+encodeURIComponent(pending));if(data&&data.token){state.token=data.token;sessionStorage.setItem("rustdesk.console.token",data.token);sessionStorage.removeItem("rustdesk.console.oidc");}}catch{}}
    }
    if(state.token){
      try{
        state.user=await request("/user/current");
        const routes=Array.isArray(state.user.route_names)?state.user.route_names:[];
        state.isAdmin=routes.includes("*");
      }catch{state.token="";sessionStorage.removeItem("rustdesk.console.token");}
    }
    if(!state.token){await getOptions();render();return;}
    if(returnToWebClient())return;
    const hash=location.hash.replace(/^#\/?/,"").split("?")[0];
    state.page=hash==="devices"&&!available(hash)?"mydevices":available(hash)?hash:"overview";
    const routeParams=new URLSearchParams((location.hash.split("?")[1]||""));
    state.query=routeParams.get("search")||"";
    render();await load();
    const entityType=routeParams.get("entity"),entityID=routeParams.get("value");
    if(["device","user"].includes(entityType)&&entityID)await openEntityDetail(entityType,entityID,entityID);
  }
  async function getOptions(){
    try{state.options=await request("/login-options");}catch{state.options={ops:[]};}
    if(state.options.need_captcha)await captcha();
  }
  async function captcha(){try{state.captcha=(await request("/captcha")).captcha;}catch{state.captcha=null;}}
  async function handleLogin(form){
    const data=Object.fromEntries(new FormData(form).entries());data.platform="web";
    const submit=form.querySelector('button[type="submit"]');submit.disabled=true;
    try{
      const result=await request("/login","POST",data);
      if(result?.type==="tfa_check"||result?.type==="email_check"){
        if(!result.secret||!result.user?.name)throw Error("双重认证请求不完整");
        state.modal={kind:"mfa-login",username:result.user.name,secret:result.secret,
          method:result.tfa_type||result.type};
        render();
        return;
      }
      if(!result||!result.token)throw Error("登录未返回访问令牌");
      state.token=result.token;sessionStorage.setItem("rustdesk.console.token",result.token);
      state.user=result;state.isAdmin=Array.isArray(result.route_names)&&result.route_names.includes("*");
      if(returnToWebClient())return;
      state.page="overview";location.hash="#/overview";render();await load();toast("登录成功");
    }catch(e){toast(e.message,true);await getOptions();render();}
    finally{submit.disabled=false;}
  }
  async function passkeyLogin(){
    if(!window.RustDeskPasskeys?.supported()){toast("请使用支持 Passkey 的 HTTPS 浏览器",true);return;}
    try{
      const session=await request("/passkey/login/begin","POST",{});
      const credential=await window.RustDeskPasskeys.get(session.options);
      const result=await request("/passkey/login/finish","POST",{
        session_id:session.session_id,credential
      });
      if(!result?.token)throw Error("服务器没有返回登录令牌");
      state.token=result.token;sessionStorage.setItem("rustdesk.console.token",result.token);
      state.user=result;state.isAdmin=Array.isArray(result.route_names)&&result.route_names.includes("*");
      if(returnToWebClient())return;
      state.page="overview";state.modal=null;location.hash="#/overview";
      render();await load();toast("Passkey 登录成功");
    }catch(err){toast(err.name==="NotAllowedError"?"已取消 Passkey 验证":err.message,true);}
  }
  async function passkeyRegister(){
    if(!window.RustDeskPasskeys?.supported()){toast("请通过 HTTPS 或 localhost 访问并使用支持 Passkey 的浏览器",true);return;}
    const name=String(document.getElementById("passkey-name")?.value||"我的设备").trim();
    if(!name||name.length>80){toast("Passkey 名称须在 1 至 80 个字符之间",true);return;}
    try{
      const session=await request("/passkey/register/begin","POST",{});
      const credential=await window.RustDeskPasskeys.create(session.options);
      await request("/passkey/register/finish","POST",{
        session_id:session.session_id,name,credential
      });
      const result=await request("/passkey/list");
      state.passkeys=result.list||[];state.passkeysError="";
      render();toast("Passkey 已添加");
    }catch(err){toast(err.name==="NotAllowedError"?"已取消 Passkey 添加":err.message,true);}
  }
  async function oidc(op){
    try{
      const result=await request("/oidc/auth","POST",{op,deviceInfo:{os:"web"},id:"webadmin",uuid:""});
      if(!result.url||!result.code)throw Error("无法启动单点登录");
      sessionStorage.setItem("rustdesk.console.oidc",result.code);
      window.location.assign(result.url);
    }catch(e){toast(e.message,true);}
  }
  async function logout(){
    try{await request("/logout","POST",{});}catch{}
    state.token="";state.user=null;state.isAdmin=false;state.devicePresence={};sessionStorage.removeItem("rustdesk.console.token");localStorage.removeItem("access_token");
    await getOptions();render();
  }
  async function load(){
    const n=++seq,k=state.page,p=pages[activeKey()];
    state.loading=true;state.error="";render();
    try{
      if(k==="overview"){
        const reqs=[request(endpoint(state.isAdmin?"devices":"mydevices")+"/list?page=1&page_size=5")];
        reqs.push(state.isAdmin?request("/user/list?page=1&page_size=1"):request("/my/address_book/list?page=1&page_size=1"));
        reqs.push(state.isAdmin?request("/group/list?page=1&page_size=1"):request("/contact/list"));
        const out=await Promise.allSettled(reqs);
        const val=i=>out[i].status==="fulfilled"?out[i].value:{};
        state.metrics={devices:val(0).total,users:val(1).total,groups:val(2).total,recent:val(0).list||[]};
      }else if(k==="contacts"){
        state.contacts=await request("/contact/list");
      }else if(k==="shared"){
        const q=new URLSearchParams({page:state.pageNo,page_size:state.pageSize,search:state.query,source_type:state.sourceType,source_id:state.sourceId});
        const order=state.sort[k];if(order?.key){q.set("sort_by",order.key);q.set("sort_dir",order.dir===-1?"desc":"asc");}
        const result=await request("/shared_devices/list?"+q.toString());
        state.rows=result.list||[];state.total=result.total||0;
        if(!state.groups.length){try{state.groups=(await request("/shared_devices/sources")).sources||[];}catch{state.groups=[];}}
      }else if(k==="push"){
        const [destinations,catalog,history]=await Promise.all([
          request("/push/channels"), request("/push/events"),
          request("/push/history?page=1&page_size=20")
        ]);
        state.pushChannels=destinations.list||[];
        state.pushEvents=catalog||[];
        state.pushHistory=history.list||[];
        state.pushTotal=history.total||0;
      }else if(k==="profile"){
        const results=await Promise.all([request("/mfa/status"),request("/mfa/email/status")]);
        state.mfa=results[0];
        state.emailVerification=results[1];
        try{
          const passkeys=await request("/passkey/list");
          state.passkeys=passkeys.list||[];state.passkeysError="";
        }catch(err){state.passkeys=[];state.passkeysError=err.message;}
      }else if(k==="mail"){
        state.mail=await request("/mail/settings");
      }else if(k==="clientBuilder"){
        const [profiles,tasks,versions]=await Promise.all([
          request("/client_builder/profiles?page=1&page_size=100"),
          request("/client_builder/tasks?page=1&page_size=50"),
          request("/client_builder/versions").catch(()=>({list:[]}))
        ]);
        state.clientProfiles=profiles.list||[];
        state.clientTasks=tasks.list||[];
        state.clientVersions=Array.isArray(versions.list)?versions.list:[];
      }else if(k==="notifications"){
        const [config,history]=await Promise.all([
          request("/notification/settings"),
          request("/notification/list?"+new URLSearchParams({page:state.pageNo,page_size:state.pageSize,
            ...(state.sort[k]?.key?{sort_by:state.sort[k].key,sort_dir:state.sort[k].dir===-1?"desc":"asc"}:{})}).toString())
        ]);
        state.notifications=config;
        state.rows=history.list||[];state.total=history.total||0;
      }else if(k==="retention"){
        state.retention=await request("/audit/retention");
      }else if(k==="settings"){
        const [settings,overview,history,failed]=await Promise.all([
          request("/settings/effective"),request("/settings/overview"),
          request("/settings/history?limit=15"),request("/settings/outbox?category="+state.settingsFailedCategory)
        ]);
        state.settings=settings;state.settingsOverview=overview;
        state.settingsHistory=history.list||[];state.settingsFailed=failed.list||[];
      }else if(k==="server"){
        state.server=await request("/config/server");
      }else if(p.endpoint){
        const q=new URLSearchParams({page:state.pageNo,page_size:state.pageSize});
        if(p.search&&state.query)q.set(p.search,state.query);
        const order=state.sort[k];if(order?.key){q.set("sort_by",order.key);q.set("sort_dir",order.dir===-1?"desc":"asc");}
        const result=await request(endpoint(activeKey())+"/list?"+q.toString());
        state.rows=result.list||[];state.total=result.total||0;
      }
    }catch(e){state.error=e.message;state.rows=[];state.total=0;}
    if(n!==seq)return;
    if(p?.columns||p?.fields)await loadPageReferences(p);
    if(n!==seq)return;
    await loadGeoIPLocations();
    if(n!==seq)return;state.loading=false;render();
  }
  function metric(name,value,ic){
    return '<div class="metric"><span class="metric-icon">'+icon(ic)+'</span><div class="metric-value">'+esc(value===undefined?"—":value)+'</div><div class="metric-label">'+esc(name)+'</div></div>';
  }
  function overview(){
    const m=state.metrics||{};
    const shortcuts=["mydevices","shared","contacts",state.isAdmin?"devices":"mygroups"];
    const quick=shortcuts.map(k=>'<button type="button" data-view="'+k+'" class="quick-card"><span class="quick-icon">'+icon(pages[k].icon,21)+'</span><strong class="quick-label">'+esc(pages[k].name)+'</strong><span class="quick-arrow">'+icon("arrow",17)+'</span></button>').join("");
    const recent=(m.recent||[]).map(x=>'<div class="status-row"><span><strong>'+entityLink("device",x.id,x.alias||x.hostname||x.id,true)+'</strong><span class="cell-sub">'+entityLink("device",x.id,x.id,false)+'</span></span><span class="pill muted">'+date(x.last_online_time)+'</span></div>').join("")||'<div class="empty">暂无设备</div>';
    return pageHeading("控制台概览","一个统一的空间，用于管理远程设备与共享关系。")+
      '<div class="metric-grid">'+metric("设备数量",state.loading?"…":m.devices,"monitor")+metric(state.isAdmin?"用户数量":"地址簿条目",state.loading?"…":m.users,"users")+metric(state.isAdmin?"用户分组":"联系人数量",state.loading?"…":m.groups===undefined?state.contacts.incoming.length:m.groups,"layers")+metric("当前账户",state.user.username,"shield")+'</div>'+
      '<div class="two-cols"><section class="panel"><div class="panel-header"><div><h2 class="panel-title">快速访问</h2><p class="panel-sub">常用管理功能</p></div></div><div class="quick-grid">'+quick+'</div></section><section class="panel"><div class="panel-header"><div><h2 class="panel-title">最近出现的设备</h2><p class="panel-sub">按最后活动时间显示</p></div><button data-view="'+(state.isAdmin?"devices":"mydevices")+'" class="btn btn-sm">查看全部</button></div><div class="tips">'+recent+'</div></section></div>';
  }
  function rowIdentifier(row){return String(row.row_id===undefined?row.id:row.row_id);}
  function rowActions(row){
    const p=pages[activeKey()];
    if(state.page==="trash")return '<button type="button" class="btn btn-sm" data-action="trash-restore" data-id="'+esc(row.row_id)+'">恢复</button>'+
      '<button type="button" class="btn btn-sm btn-danger" data-action="trash-purge" data-id="'+esc(row.row_id)+'">彻底删除</button>';
    if(state.page==="invitations"){
      if(row.accepted_at)return '<span class="pill good">已接受</span>';
      if(row.revoked_at)return '<span class="pill muted">已撤销</span>';
      return '<button class="btn btn-sm btn-danger" data-action="invitation-revoke" data-id="'+esc(row.id)+'">撤销邀请</button>';
    }
    if(state.page==="connections")return '<button type="button" class="btn btn-sm" data-action="audit-note" data-id="'+esc(row.id)+'">编辑备注</button>';
    if(state.page==="notifications"){
      return row.status==="failed"&&canPageAction("notifications","edit")?
        '<button type="button" class="btn btn-sm" data-action="notification-retry" data-id="'+esc(row.id)+'">重新发送</button>':"";
    }
    if(state.page==="apiTokens"){
      return row.revoked_at?'<span class="pill muted">已撤销</span>':
        '<button type="button" class="btn btn-sm btn-danger" data-action="revoke-api-token" data-id="'+esc(row.id)+'">撤销令牌</button>';
    }
    if(["activeSessions","myactiveSessions"].includes(state.page))return '<button type="button" class="btn btn-sm btn-danger" data-action="session-disconnect" data-id="'+esc(row.conn_id)+'" data-device="'+esc(row.device_id)+'">结束连接</button>';
    if(["sessions","mysessions"].includes(state.page))return '<button type="button" class="btn btn-sm btn-danger" data-action="session-revoke" data-id="'+esc(row.id)+'">结束登录</button>';
    if(state.page==="deployed")return '<button type="button" class="btn btn-sm" data-action="deployment-toggle" data-id="'+esc(row.id)+'">'+(row.status==="approved"?"撤销部署":"恢复部署")+"</button>";
    if(p.readonly)return "";
    const id=esc(rowIdentifier(row));
    let out=state.page==="mydevices"?'<button type="button" class="btn btn-sm btn-primary" data-action="device-connect" data-device-id="'+esc(row.id)+'" data-device-name="'+esc(row.alias||row.hostname||"")+'">连接设备</button>':"";
    const editable=pages[state.page]?.admin?canPageAction(state.page,"edit"):true;
    if(editable)out+='<button class="btn btn-sm" data-action="edit" data-id="'+id+'">编辑</button>';
    if(state.page==="devices"&&(state.isAdmin||hasPrivilege("devices.transfer"))){
      out+='<button class="btn btn-sm" data-action="transfer-device" data-id="'+id+'">转移设备</button>';
    }
    if(state.page==="users"&&hasPrivilege("users.edit_password")){
      out+='<button class="btn btn-sm" data-action="reset-password" data-id="'+id+'">重设密码</button>';
    }
    if(state.page==="users"&&hasPrivilege("users.manage_2fa")){
      out+='<button class="btn btn-sm" data-action="reset-user-mfa" data-id="'+id+'">清除双重认证</button>';
    }
    if(state.page==="groups"||state.page==="mygroups"){out+='<button class="btn btn-sm" data-action="members" data-id="'+id+'">成员</button>';if(row.type===3)out+='<button class="btn btn-sm" data-action="anchors" data-id="'+id+'">公共成员</button>';}
    const deletable=pages[state.page]?.admin?canPageAction(state.page,"delete"):true;
    if(deletable)out+='<button class="btn btn-sm btn-danger" data-action="delete" data-id="'+id+'">删除</button>';
    return out;
  }
  function tableColumnWidth(key,label,page){
    const deviceIDPages=new Set(["devices","mydevices","shared","addressbooks","trash"]);
    const widths={
      id:deviceIDPages.has(page)?164:80,
      device_id:200,peer_id:176,from_peer:176,
      device_uuid:176,conn_id:145,conn_audit_ref:190,
      direction:172,type:128,rule:136,status:130,enabled:126,
      name:180,alias:162,hostname:168,nickname:164,
      username:170,owner_username:178,from_account:178,to_account:178,
      account:160,email:236,sender_email:226,
      from_name:174,from_device_name:174,to_device_name:174,
      from_user_id:160,to_user_id:160,owner_user_id:160,
      user_id:136,group_id:140,collection_id:142,policy_id:158,
      client_ip:192,last_online_ip:192,source_ip:192,ip:192,
      created_at:190,deleted_at:190,updated_at:190,last_online_time:190,
      last_used_at:190,expires_at:190,completed_at:190,delivered_at:190,
      os:140,platform:160,method:116,primary_auth:152,two_factor:148,
      path:270,info:272,note:250,remark:240,description:260,
      sources:230,issuer:262,permissions:250,
      id_whitelist:250,ip_whitelist:250,source_type:164,target_type:164,
      target_id:160,source_id:160,scope_type:166,scope_id:145,
      event_type:178,recipients:220,last_error:250,
      auto_group_ids:190,trust_duration_days:195,
      oauth_type:138,op:172,typ:150
    };
    const labelWidth=Array.from(label).reduce((sum,char)=>
      sum+(char.charCodeAt(0)>255?13:7),0)+36;
    return Math.max(widths[key]||144,Math.min(300,labelWidth));
  }
  function table(p,rows){
    if(state.loading)return '<div class="loading-table"><span class="spinner"></span>正在读取数据…</div>';
    if(state.error)return '<div class="empty">'+icon("info",32)+'<h3>读取数据失败</h3><p>'+esc(state.error)+'</p></div>';
    if(!rows.length)return '<div class="empty">'+icon("info",32)+'<h3>暂无记录</h3><p>请调整筛选条件或创建新的记录。</p></div>';
    const allCols=p.columns||[];
    const actions=rows.map(row=>rowActions(row));
    const viewport=Number(window.innerWidth)||1440;
    const availableWidth=Math.max(630,viewport-(viewport>1190?252:222)-(viewport>1190?64:40));
    const widths=allCols.map(([key,label])=>{
      const width=tableColumnWidth(key,label,state.page);
      return Math.max(104,Math.round(width*.76));
    });
    const priorities=allCols.map(([key],index)=>{
      const important=new Set(["id","device_id","peer_id","alias","hostname","username","nickname","name",
        "last_online_time","last_online_ip","ip","status","group_id","email","created_at","last_seen_at",
        "conn_id","owner_username","account","requested_by","target_id","address","client","platform",
        "active","current","expired_at","login_ip"]);
      const participantPages=new Set(["myconnections","connections","mytransfers","transfers"]);
      const participantColumns=["from_peer","peer_id","from_account","to_account","direction","created_at"];
      const participantRank=participantPages.has(state.page)?participantColumns.indexOf(key):-1;
      return (index===0?1000:participantRank>=0?300-participantRank*12:important.has(key)?120:40)-(index*.1);
    });
    const ordered=allCols.map((_,i)=>i).sort((a,b)=>priorities[b]-priorities[a]);
    const selected=new Set(),actionWidth=158;
    let needed=actionWidth;
    if(state.showAllColumns[state.page]){
      allCols.forEach((_,i)=>selected.add(i));
    }else{
      for(const i of ordered){
        if(selected.size>0&&needed+widths[i]>availableWidth)continue;
        selected.add(i);needed+=widths[i];
      }
      if(!selected.size)selected.add(0);
    }
    const visible=allCols.filter((_,i)=>selected.has(i));
    const hidden=allCols.filter((_,i)=>!selected.has(i));
    const hasActions=actions.some(Boolean)||hidden.length>0;
    const actionSize=hasActions?actionWidth:0;
    const tableMinWidth=visible.reduce((sum,col)=>sum+widths[allCols.indexOf(col)],actionSize);
    const order=state.sort[state.page]||{};
    const colgroup='<colgroup>'+visible.map(col=>'<col style="width:'+widths[allCols.indexOf(col)]+'px">').join("")+
      (hasActions?'<col style="width:'+actionSize+'px">':"")+'</colgroup>';
    const th=visible.map(([key,label])=>'<th scope="col" data-column="'+esc(key)+'" aria-sort="'+
      (order.key===key?(order.dir===1?"ascending":"descending"):"none")+
      '" title="'+esc(label)+'"><button type="button" class="sort-heading" data-action="sort-table" data-column="'+esc(key)+'">'+
      esc(label)+'<span aria-hidden="true">'+(order.key===key?(order.dir===1?"▲":"▼"):"↕")+'</span></button></th>').join("");
    const multiline=new Set(["os","platform","note","remark","info","description","sources","permissions",
      "id_whitelist","ip_whitelist","last_error","last_online_ip","ip","source_ip","client_ip","login_ip",
      "email","hostname","issuer","path","device_uuid","from_account","to_account"]);
    const trs=rows.map((row,index)=>{
      const cells=visible.map(([key])=>{
        const first=key===allCols[0][0];
        return '<td data-column="'+esc(key)+'" class="'+(multiline.has(key)?"cell-multiline ":"")+
          (first?"cell-primary":"")+'" title="'+esc(txt(row[key]))+'"><span class="'+(first?"cell-title":"")+
          '">'+colValue(row[key],key,row)+'</span></td>';
      }).join("");
      const detail=hidden.length?'<button type="button" class="btn btn-sm" data-action="table-details" data-row="'+index+'" aria-expanded="false">查看详情</button>':"";
      const main='<tr>'+cells+(hasActions?'<td class="cell-actions"><div class="cell-action-list">'+actions[index]+detail+'</div></td>':"")+'</tr>';
      const expanded=hidden.length?'<tr class="table-detail-row" data-details-row="'+index+'" hidden><td colspan="'+
        (visible.length+(hasActions?1:0))+'"><dl class="table-detail-grid">'+hidden.map(([key,label])=>
          '<div><dt>'+esc(label)+'</dt><dd>'+colValue(row[key],key,row)+'</dd></div>').join("")+'</dl></td></tr>':"";
      return main+expanded;
    }).join("");
    const mobile=rows.map((row,index)=>'<article class="mobile-item"><div class="mobile-title">'+
      colValue(row[allCols[0][0]],allCols[0][0],row)+'</div><dl>'+allCols.slice(1).map(([key,label])=>
      '<dt>'+esc(label)+'</dt><dd>'+colValue(row[key],key,row)+'</dd>').join("")+
      '</dl>'+(actions[index]?'<div class="mobile-actions">'+actions[index]+'</div>':"")+
      '</article>').join("");
    const mobileSort='<div class="mobile-sort"><label for="mobile-sort-column">按列排序</label><select id="mobile-sort-column" aria-label="排序字段">'+
      allCols.map(([key,label])=>'<option value="'+esc(key)+'" '+((state.sort[state.page]?.key||allCols[0][0])===key?"selected":"")+'>'+esc(label)+'</option>').join("")+
      '</select><button type="button" class="btn btn-sm" data-action="mobile-sort-dir">'+(state.sort[state.page]?.dir===-1?"降序 ▼":"升序 ▲")+'</button></div>';
    const expandedActions=hidden.length?'<button type="button" class="btn btn-sm" data-action="table-columns">'+
      (state.showAllColumns[state.page]?"按窗口宽度显示":"显示全部列")+'</button>':state.showAllColumns[state.page]?
      '<button type="button" class="btn btn-sm" data-action="table-columns">按窗口宽度显示</button>':"";
    const sortSelect=hidden.length?'<label class="table-sort-label">排序字段 <select id="table-sort-column" aria-label="排序字段">'+
      allCols.map(([key,label])=>'<option value="'+esc(key)+'" '+(order.key===key?"selected":"")+'>'+esc(label)+'</option>').join("")+'</select></label>':"";
    const hint='<div class="table-scroll-hint"><span>点击列标题排序；隐藏的字段可通过查看详情读取。</span>'+sortSelect+expandedActions+'</div>';
    return '<div class="table-scroll" role="region" aria-label="'+esc(p.name||"数据")+
      '数据表" tabindex="0"><table class="data-table" style="--table-min-width:'+tableMinWidth+'px">'+
      colgroup+'<thead><tr>'+th+(hasActions?'<th scope="col">操作</th>':"")+
      '</tr></thead><tbody>'+trs+'</tbody></table></div>'+hint+
      mobileSort+'<div class="mobile-list">'+mobile+'</div>';
  }
  function pagination(){
    const pagesCount=Math.max(1,Math.ceil(state.total/state.pageSize));
    return '<div class="pager"><span>第 '+state.pageNo+' / '+pagesCount+' 页 · 共 '+state.total+' 项</span><div class="pager-buttons"><button data-action="prev" '+(state.pageNo===1?"disabled":"")+' aria-label="上一页">‹</button><button data-action="next" '+(state.pageNo>=pagesCount?"disabled":"")+' aria-label="下一页">›</button></div></div>';
  }
  function genericList(){
    const p=pages[activeKey()];
    const add=p.readonly||p.noCreate||state.page==="devices"&&!canPageAction("devices","create")||
      (p.admin&&!canPageAction(state.page,"create"))?"":'<button class="btn btn-primary" data-action="create">'+icon("plus",16)+'新增'+esc(p.name)+'</button>';
    const search='<div class="table-toolbar"><div class="search">'+icon("search",16)+'<input id="table-search" aria-label="搜索" placeholder="'+(p.search?"按关键词搜索…":"筛选本页内容…")+'" value="'+esc(state.query)+'"></div><span class="toolbar-meta">共 '+state.total+' 项</span></div>';
    const rows=p.search?state.rows:state.rows.filter(row=>!state.query||JSON.stringify(row).toLowerCase().includes(state.query.toLowerCase()));
    const isBook = state.page==="addressbooks";
    const tabs = isBook?'<div style="display:flex;gap:8px;margin-bottom:18px;flex-wrap:wrap"><button class="btn '+(state.bookTab==="entries"?"btn-primary":"")+'" data-action="book-entries">设备条目</button><button class="btn '+(state.bookTab==="collections"?"btn-primary":"")+'" data-action="book-collections">地址簿分类</button></div>':"";
    const description=state.page==="accessRules"?
      "可选的服务器连接访问规则。启用 hbbs 严格模式后，将按这些规则判断远程连接。未配置时保持原来的连接方式。":
      state.page==="controlRoles"?
      "为远程会话设置键盘、剪贴板、文件传输、终端等权限。未指定的操作沿用设备自身设置。":
      state.page==="controlAssignments"?
      "给设备、设备所有者或设备组分配控制角色。单台设备的设置优先级最高。":
      state.page==="relayNodes"?"配置多个 hbbr 节点及各自优先服务的 IP 网络。hbbs 会在启用自动同步后更新中继列表，并保留节点不可用时的替代选择。":isBook?"在一个页面中管理地址簿分类及其中的设备，原有数据将继续保留。":state.page==="mygroups"?"创建和管理您拥有的群组。其他人的群组需要由其所有者或系统管理员管理。":state.page==="trash"?"找回误删的设备条目，也可以永久删除指定条目。删除整个地址簿分类时将永久清除该分类下的记录。":state.page==="apiTokens"?"创建可随时撤销的管理令牌，用于设备部署、只读查询和自动化管理。权限名称及接口请参阅 API 文档。":state.page==="policies"?"为自行管理的 RustDesk 客户端制定设备 ID 白名单、IP 白名单、WebRTC 和可信设备有效期设置。":state.page==="policyAssignments"?"选择设备并分配安全策略。支持策略读取接口的客户端可自动应用设置。":state.page.startsWith("my")&&p.readonly?"这里只显示与您本人或您名下设备有关的记录。":"查看、编辑和管理相关数据。";
    const exportKinds={loginlogs:"login_log",connections:"audit_conn",transfers:"audit_file",alarms:"audit_alarm",operations:"audit_operation"};
    const exportBtn=exportKinds[state.page]?'<button type="button" class="btn" data-action="audit-export" data-id="'+exportKinds[state.page]+'">筛选并导出 CSV</button>':
      state.page==="devices"&&canPageAction("devices","view")?
        '<button type="button" class="btn" data-action="devices-export">导出设备清单 CSV</button>':"";
    const csvActions=isBook?
      '<button type="button" class="btn" data-view="trash">查看回收站</button>'+
      '<button type="button" class="btn" data-action="csv-import">导入 CSV</button>'+
      '<button type="button" class="btn" data-action="csv-export">导出 CSV</button>':"";
    const trashBatchActions=state.page==="trash"?
      '<button type="button" class="btn" data-action="trash-batch-restore">批量恢复</button>'+
      '<button type="button" class="btn" data-action="trash-batch-purge">批量永久删除</button>':"";
    const deploymentBatchActions=state.page==="deployed"?
      '<button type="button" class="btn" data-action="deployment-batch-approve">批量批准部署</button>'+
      '<button type="button" class="btn" data-action="deployment-batch-revoke">批量撤销部署</button>':"";
    const sharingBatchActions=state.page==="myRules"?
      '<button type="button" class="btn" data-action="rules-batch-share">批量共享地址簿</button>'+
      '<button type="button" class="btn" data-action="rules-batch-revoke">批量撤销共享</button>':"";
    return pageHeading(isBook?"地址簿":p.name,description,add+exportBtn+csvActions+trashBatchActions+deploymentBatchActions+sharingBatchActions)+tabs+'<section class="panel">'+search+table(p,rows)+pagination()+'</section>';
  }
  function contacts(){
    const v=state.contacts||{outgoing:[],incoming:[]};
    function accountRows(rows,canDelete){
      if(state.loading)return '<div class="loading-table"><span class="spinner"></span>加载中…</div>';
      if(!rows.length)return '<div class="empty"><h3>暂无联系人</h3><p>'+(canDelete?"您还没有主动向其他用户公开设备。":"目前没有用户向您公开设备。")+'</p></div>';
      return rows.map(x=>'<div class="status-row"><span><strong>'+entityLink("user",x.user_id||x.id||x.username,x.nickname||x.username)+'</strong><span class="cell-sub">@'+entityLink("user",x.user_id||x.id||x.username,x.username)+' · 用户 ID '+esc(x.user_id)+'</span></span>'+(canDelete?'<button class="btn btn-sm btn-danger" data-action="contact-delete" data-id="'+esc(x.user_id)+'">撤销共享</button>':'<button class="btn btn-sm" data-action="contact-devices" data-id="'+esc(x.user_id)+'">查看设备</button>')+'</div>').join("");
    }
    return pageHeading("联系人","添加联系人即表示向对方开放您名下的所有设备，对方无须确认。",
      '<button class="btn" data-action="contact-batch-add">批量添加联系人</button>'+
      '<button class="btn" data-action="contact-batch-remove">批量撤销共享</button>')+
      '<div class="two-cols"><section class="panel"><div class="panel-header"><div><h2 class="panel-title">我向谁共享设备</h2><p class="panel-sub">只有您主动添加的人才能通过联系人关系查看您的设备</p></div></div><div class="tips">'+accountRows(v.outgoing||[],true)+'</div><form id="contact-form" class="table-toolbar" style="gap:10px"><div class="field" style="flex:1 1 230px;min-width:0"><label>搜索并选择用户</label>'+pickerField("user_id","",true,{})+'</div><button type="submit" class="btn btn-primary">'+icon("plus",16)+'添加联系人</button></form></section><section class="panel"><div class="panel-header"><div><h2 class="panel-title">向我共享设备的人</h2><p class="panel-sub">这些用户已主动向您开放他们的设备</p></div></div><div class="tips">'+accountRows(v.incoming||[],false)+'</div></section></div>';
  }
  function shared(){
    const groups=(state.groups||[]);
    const options='<option value="all">全部共享设备</option><option value="contact">联系人共享</option><option value="group">群组共享</option>';
    const ids=groups.filter(x=>state.sourceType==="all"||x.type===state.sourceType).map(x=>'<option value="'+esc(x.type+":"+x.id)+'" '+(x.type===state.sourceType&&String(x.id)===String(state.sourceId)?"selected":"")+'>'+esc(x.type==="contact"?"联系人 · "+x.name:"群组 · "+x.name)+'</option>').join("");
    const search='<div class="table-toolbar"><div class="search">'+icon("search",16)+'<input id="table-search" placeholder="搜索设备 ID、备注或主机名" value="'+esc(state.query)+'"></div><select id="shared-source-type" class="btn" aria-label="共享类型">'+options.replace('value="'+state.sourceType+'"','value="'+state.sourceType+'" selected')+'</select><select id="shared-source-id" class="btn" aria-label="共享来源"><option value="0">所有来源</option>'+ids+'</select></div>';
    return pageHeading("共享设备","汇总联系人、共享组和公共锚点组公开给您的设备。同一设备只显示一次，并标明所有共享来源。",'<button class="btn btn-primary" data-action="webclient">'+icon("monitor",16)+'打开 WebClient v2</button>')+
      '<section class="panel">'+search+table({...pages.shared},state.rows)+pagination()+'</section>';
  }
  const serverCommandCatalog=[
    ["id-help","ID 服务器","查看服务器帮助","查看当前服务器支持的管理功能。","21115","h","none"],
    ["id-relay-read","ID 服务器","查看中继服务器列表","读取 ID 服务器当前配置的中继地址。","21115","relay-servers","none"],
    ["id-relay-write","ID 服务器","设置中继服务器列表","填写中继服务器主机名或地址，多个地址用英文逗号分隔。","21115","relay-servers","servers"],
    ["id-block-read","ID 服务器","查看已封禁的 IP","查询 ID 服务器当前的 IP 封禁记录。","21115","ip-blocker","none"],
    ["id-block-add","ID 服务器","查询指定 IP 的封禁记录","查看指定 IP 的封禁次数及封禁时长。","21115","ip-blocker","ip"],
    ["id-block-remove","ID 服务器","清除指定 IP 的封禁记录","填写需要清除封禁记录的 IP 地址。","21115","ip-blocker","ip-remove"],
    ["id-changes-read","ID 服务器","查看 IP 变化记录","查看客户端 IP 地址变化情况。","21115","ip-changes","none"],
    ["id-changes-item","ID 服务器","查询设备 IP 变化","搜索设备并查看其 IP 地址变化记录。","21115","ip-changes","identifier"],
    ["id-changes-remove","ID 服务器","清除设备 IP 变化记录","搜索设备并清除该设备已记录的 IP 变化信息。","21115","ip-changes","identifier-remove"],
    ["id-must-login-read","ID 服务器","查询客户端登录要求","查看客户端是否需要登录账户才能连接。","21115","must-login","none"],
    ["id-must-login-on","ID 服务器","启用客户端登录要求","要求客户端登录账户。","21115","must-login","fixed-y"],
    ["id-must-login-off","ID 服务器","关闭客户端登录要求","允许按照当前服务端策略连接。","21115","must-login","fixed-n"],
    ["id-reload-geo","ID 服务器","更新地区数据库","重新读取服务端已配置的地区数据库。","21115","reload-geo","none"],
    ["id-punch","ID 服务器","查看最近穿透请求","查看服务器记录的最近连接尝试。","21115","punch-requests","none"],
    ["id-relay-state","ID 服务器","查询强制中继状态","读取是否已要求所有连接使用中继。","21115","always-use-relay","none"],
    ["id-relay-on","ID 服务器","启用强制中继","要求所有远程连接使用中继服务器。","21115","always-use-relay","fixed-y"],
    ["id-relay-off","ID 服务器","关闭强制中继","允许服务器按正常条件选择直接连接或中继。","21115","always-use-relay","fixed-n"],
    ["id-geo","ID 服务器","测试地址所属地区","分别填写两个 IP 地址，查看服务器地区识别结果。","21115","test-geo","ip-pair"],
    ["relay-help","中继服务器","查看服务器帮助","查看中继服务器支持的管理功能。","21117","h","none"],
    ["relay-blacklist-read","中继服务器","查看全部限速名单 IP","查看中继服务器当前限速名单中的全部 IP 地址。","21117","blacklist","none"],
    ["relay-blacklist-check","中继服务器","查询限速名单","填写 IP 地址，检查该地址是否在限速名单中。","21117","blacklist","ip"],
    ["relay-blacklist-add","中继服务器","添加限速名单","把指定 IP 加入限速名单，超过阈值时会对其传输速度进行限制。","21117","blacklist-add","ip"],
    ["relay-blacklist-remove","中继服务器","移除限速名单","将指定 IP 地址移出中继服务器限速名单。","21117","blacklist-remove","ip"],
    ["relay-blocklist-read","中继服务器","查看全部阻止列表 IP","查看中继服务器当前阻止列表中的全部 IP 地址。","21117","blocklist","none"],
    ["relay-blocklist-check","中继服务器","查询阻止列表","填写 IP 地址，检查该地址是否在阻止列表中。","21117","blocklist","ip"],
    ["relay-blocklist-add","中继服务器","添加阻止列表","把指定 IP 地址加入阻止列表。","21117","blocklist-add","ip"],
    ["relay-blocklist-remove","中继服务器","移除阻止列表","从阻止列表移除指定 IP 地址。","21117","blocklist-remove","ip"],
    ["relay-threshold-read","中继服务器","查询降级阈值","查看当前中继降级阈值。","21117","downgrade-threshold","none"],
    ["relay-threshold-set","中继服务器","设置降级阈值","填写 0 到 1 之间的比例，例如 0.66。超过此比例后，服务器会按条件检查是否需要限制连接速度。","21117","downgrade-threshold","number"],
    ["relay-check-read","中继服务器","查询降级检查间隔","查看启动降级检查的时间间隔。","21117","downgrade-start-check","none"],
    ["relay-check-set","中继服务器","设置降级检查间隔","填写大于 0 的整数秒数。","21117","downgrade-start-check","number"],
    ["relay-speed-read","中继服务器","查询速度限制","查看当前中继传输速度限制。","21117","limit-speed","none"],
    ["relay-speed-set","中继服务器","设置速度限制","填写传输速度上限，单位 Mb/s。","21117","limit-speed","number"],
    ["relay-total-read","中继服务器","查询总带宽","查看当前中继服务器的总带宽设置。","21117","total-bandwidth","none"],
    ["relay-total-set","中继服务器","设置总带宽","填写总带宽上限，单位 Mb/s。","21117","total-bandwidth","number"],
    ["relay-single-read","中继服务器","查询单连接带宽","查看单条远程连接的带宽上限。","21117","single-bandwidth","none"],
    ["relay-single-set","中继服务器","设置单连接带宽","填写单条连接的带宽上限，单位 Mb/s。","21117","single-bandwidth","number"],
    ["relay-usage","中继服务器","查看中继使用情况","查看中继服务器统计的当前使用情况。","21117","usage","none"]
  ].map(([id,section,title,description,target,cmd,mode])=>({id,section,title,description,target,cmd,mode}));
  function commandInputs(command){
    const mode=command.mode;
    if(mode==="none"||mode.startsWith("fixed-"))return '<p class="server-command-help">此操作无需填写参数。</p>';
    const ip='例如：192.0.2.10 或 2001:db8::1';
    if(mode==="ip-pair")return '<div class="form-grid"><div class="field"><label for="command-ip1">第一个 IP 地址</label><input id="command-ip1" name="ip1" type="text" placeholder="'+ip+'" spellcheck="false" required></div><div class="field"><label for="command-ip2">第二个 IP 地址</label><input id="command-ip2" name="ip2" type="text" placeholder="'+ip+'" spellcheck="false" required></div></div>';
    const title=mode==="servers"?"中继服务器地址（逗号分隔）":mode==="number"?"数值":mode==="identifier"?"设备 ID":"IP 地址";
    const placeholder=mode==="servers"?"例如：relay.example.com:21117,relay2.example.com:21117":mode==="number"?(command.cmd==="downgrade-threshold"?"例如：0.66":"填写大于 0 的数值"):mode==="identifier"?"例如：123456789":ip;
    if(mode==="identifier"||mode==="identifier-remove")return '<div class="field"><label>选择设备</label>'+pickerField("device_id","",true,{},"command-option")+'</div>';
    const attr=mode==="number"?(command.cmd==="downgrade-threshold"?'type="number" min="0.0001" max="1" step="any"':command.cmd==="downgrade-start-check"?'type="number" min="1" max="1000000000" step="1"':'type="number" min="0.0001" max="1000000000" step="any"'):'type="text" spellcheck="false"';
    return '<div class="field"><label for="command-option">'+title+'</label><input id="command-option" name="option" '+attr+' placeholder="'+placeholder+'" required maxlength="1800"></div>';
  }
  function serverCommandCategory(command){
    const id=command.id;
    if(id.includes("help")||id.includes("usage")||id.includes("punch")||id.includes("geo"))return "查看信息";
    if(id.includes("block")||id.includes("blacklist")||id.includes("changes"))return "访问与网络记录";
    if(command.cmd==="relay-servers"||command.cmd==="always-use-relay"||command.cmd==="must-login")return "连接策略";
    return "性能与带宽";
  }
  function renderServerResult(result,command){
    if(!result)return '<div class="server-result-empty">'+icon("info",24)+
      '<strong>尚未查询</strong><p>选择左侧功能并执行，查询结果将在这里展示。</p></div>';
    const summary='<p class="server-result-summary">'+esc(result.summary||"服务器已处理该请求")+'</p>';
    if(result.kind==="boolean")return summary+'<div class="server-bool-state"><span class="pill '+(result.active?"good":"muted")+'">'+
      (result.active?"已启用或已匹配":"已停用或未匹配")+'</span></div>';
    if(result.kind==="metric")return summary+'<div class="server-metric">'+esc(String(result.value??"—"))+
      '<small>'+esc(result.unit||"")+'</small></div>';
    if(result.kind==="value")return summary+'<div class="server-value">'+esc(result.value||"—")+'</div>';
    if(["list","table","usage"].includes(result.kind)){
      const rows=Array.isArray(result.rows)?result.rows:[];
      if(!rows.length)return summary+'<p class="server-result-empty">没有符合条件的记录。</p>';
      const candidates=result.kind==="usage"?[["address","连接 IP"],["seconds","连接时长（秒）"],["megabytes","流量（MB）"],
        ["peak_kbps","最高速率（kb/s）"],["average_kbps","平均速率（kb/s）"],["current_kbps","当前速率（kb/s）"],["detail","其他信息"]]:
        result.kind==="table"?[["name","设备或 IP"],["detail","记录详情"]]:
        [["name","功能名称"],["time","记录时间"],["address","服务器或 IP 地址"],["command","对应命令"],["detail","内容"]];
      const columns=candidates.filter(([key])=>rows.some(item=>item[key]!==undefined&&item[key]!==""));
      return summary+'<div class="server-result-table-scroll"><table class="server-result-table"><thead><tr>'+
        columns.map(([,label])=>'<th>'+esc(label)+'</th>').join("")+'</tr></thead><tbody>'+
        rows.map(row=>'<tr>'+columns.map(([key])=>'<td>'+esc(row[key]||"—")+'</td>').join("")+'</tr>').join("")+
        '</tbody></table></div>';
    }
    return summary;
  }
  function server(){
    const d=state.server||{},tab=state.serverTab;
    const labels=[["ID 服务器",d.id_server,"客户端登记、连接协商"],["中继服务器",d.relay_server,"无法直连时转发数据"],["API 服务",d.api_server,"用户登录与设备管理"]];
    const cards=labels.map(([title,address,desc])=>
      '<div class="server-overview-card"><span>'+esc(title)+'</span><strong>'+showServerAddress(address)+'</strong><small>'+esc(desc)+'</small></div>').join("");
    const tabs=[["overview","服务器概况"],["id","ID 服务器"],["relay","中继服务器"]].map(([key,label])=>
      '<button type="button" class="server-tab '+(tab===key?"active":"")+'" role="tab" aria-selected="'+(tab===key)+
      '" data-action="server-tab" data-tab="'+key+'">'+esc(label)+'</button>').join("");
    const heading=pageHeading("服务器管理","查看服务器配置、连接策略及运行统计，并通过表单执行管理操作。");
    const nav='<div class="server-tabs" role="tablist" aria-label="服务器管理分类">'+tabs+'</div>';
    if(tab==="overview"){
      return heading+nav+'<section class="panel"><div class="panel-header"><div><h2 class="panel-title">服务器概况</h2>'+
        '<p class="panel-sub">服务器地址来自当前配置。</p></div></div><div class="server-overview-grid">'+cards+'</div></section>'+
        '<div class="server-overview-quick"><button class="server-quick-action" type="button" data-action="server-tab" data-tab="id">'+
        icon("network",24)+'<strong>ID 服务器管理</strong><span>中继列表、连接策略、地址记录与网络诊断</span>'+icon("arrow",17)+'</button>'+
        '<button class="server-quick-action" type="button" data-action="server-tab" data-tab="relay">'+
        icon("shield",24)+'<strong>中继服务器管理</strong><span>IP 列表、带宽限制和连接使用统计</span>'+icon("arrow",17)+'</button></div>';
    }
    const section=tab==="id"?"ID 服务器":"中继服务器";
    const commands=serverCommandCatalog.filter(cmd=>cmd.section===section);
    const selected=commands.find(cmd=>cmd.id===state.commandSelected)||commands[0];
    const groups=[...new Set(commands.map(serverCommandCategory))].map(category=>
      '<section class="server-command-group"><h3>'+esc(category)+'</h3><div class="server-command-list">'+
      commands.filter(cmd=>serverCommandCategory(cmd)===category).map(cmd=>
        '<button type="button" class="server-command-item '+(cmd.id===selected.id?"selected":"")+
        '" data-action="server-select" data-id="'+esc(cmd.id)+'" aria-pressed="'+(cmd.id===selected.id)+'">'+
        '<strong>'+esc(cmd.title)+'</strong><small>'+esc(cmd.description)+'</small></button>').join("")+
      '</div></section>').join("");
    const result=state.serverResults[selected.id]||null;
    return heading+nav+'<div class="server-management-layout">'+
      '<aside class="panel server-command-menu"><div class="panel-header"><div><h2 class="panel-title">管理功能</h2>'+
      '<p class="panel-sub">'+esc(section)+' · 选择需要查看或修改的项目</p></div></div>'+
      '<div class="server-command-menu-body">'+groups+'</div></aside>'+
      '<div class="server-management-workspace"><section class="panel server-editor" id="server-command-panel">'+
      '<div class="panel-header"><div><h2 class="panel-title">'+esc(selected.title)+'</h2>'+
      '<p class="panel-sub">'+esc(selected.description)+'</p></div></div>'+
      '<form id="command-form" class="server-editor-body">'+commandInputs(selected)+
      '<div class="server-submit-row"><button class="btn btn-primary" type="submit">'+
      (selected.mode==="none"||selected.mode==="identifier"||selected.mode==="ip"&&["blacklist","blocklist","ip-blocker"].includes(selected.cmd)?"查询":"执行操作")+
      '</button></div></form></section>'+
      '<section class="panel server-result-panel"><div class="panel-header"><div><h2 class="panel-title">查询与操作结果</h2>'+
      '<p class="panel-sub">'+esc(selected.title)+'</p></div></div>'+
      '<div class="server-result" id="server-result" role="status">'+
      (state.serverError?'<div class="warning">'+esc(state.serverError)+'</div>':renderServerResult(result,selected))+
      '</div></section></div></div>';
  }
  const cleanupKinds=[
    ["all","全部记录类别"],["connection","远程连接记录"],["file","文件传输记录"],
    ["alarm","安全告警"],["login","用户登录记录"],["operation","管理操作记录"],
    ["disconnect","结束连接记录"],["settings_history","设置修改历史"],
    ["push_sent","个人推送成功记录"],["push_failed","个人推送失败记录"],
    ["notification_sent","管理员邮件成功记录"],["notification_failed","管理员邮件失败记录"],
    ["push_outbox_failed","个人推送事件处理失败记录"],
    ["notification_outbox_failed","管理员邮件事件处理失败记录"],
    ["server_log","服务器运行日志文件"]
  ];
  function retentionPage(){
    const data=state.retention||{};
    const fields=[["connection_days","远程连接记录",0],["file_days","文件传输记录",0],
      ["alarm_days","安全告警",0],["login_days","用户登录记录",0],["operation_days","管理操作记录",0],
      ["disconnect_days","结束连接记录",0],["settings_history_days","设置修改历史",0],
      ["push_sent_days","个人推送：发送成功",30],["push_failed_days","个人推送：发送失败",90],
      ["notification_sent_days","管理员邮件：发送成功",30],["notification_failed_days","管理员邮件：发送失败",90],
      ["push_outbox_failed_days","个人推送：事件处理失败",0],
      ["notification_outbox_failed_days","管理员邮件：事件处理失败",0],
      ["server_log_days","服务器运行日志归档文件",30]];
    const inputs=fields.map(([key,label,defaultDays])=>
      '<div class="field"><label for="'+key+'">'+label+'</label><input id="'+key+'" name="'+key+'" type="number" min="0" max="3650" step="1" required value="'+esc(data[key]??defaultDays)+'"><small>单位：天。填写 0 表示永久保留。</small></div>'
    ).join("");
    const request=state.cleanupRequest;
    const kindOptions=cleanupKinds.map(([key,label])=>
      '<option value="'+key+'" '+(request.kind===key?'selected':'')+'>'+esc(label)+'</option>').join("");
    const modes=[["retention","按照已保存的保留期限"],["older","删除早于指定天数的记录"],["all","删除所选类别的全部历史记录"]];
    const modeOptions=modes.map(([value,label])=>
      '<option value="'+value+'" '+(request.mode===value?'selected':'')+'>'+esc(label)+'</option>').join("");
    const preview=state.cleanupPreview;
    const rows=preview?.items?.filter(item=>item.eligible>0)||[];
    const previewHTML=!preview?'':('<div class="cleanup-results" role="status"><strong>预计清理 '+esc(preview.total??0)+' 项历史数据</strong>'+
      (rows.length?'<div class="cleanup-result-list">'+rows.map(item=>
        '<div class="status-row"><span>'+esc(item.name)+'</span><strong>'+esc(item.eligible)+(item.kind==="server_log"?" 个文件":" 条")+'</strong></div>'
      ).join("")+'</div>':'<p>当前没有符合所选条件的历史数据。</p>')+'</div>');
    return pageHeading("日志保留设置","为各类历史记录设置自动清理期限，也可以按类别手动清理。")+
      '<div class="retention-layout"><section class="panel"><div class="panel-header"><div><h2 class="panel-title">自动清理期限</h2>'+
      '<p class="panel-sub">服务启动后及此后每 24 小时检查一次。设置保存后立即生效，无须重新启动。</p></div></div>'+
      '<form id="retention-form" class="tips"><div class="retention-fields">'+inputs+'</div>'+
      '<button type="submit" class="btn btn-primary" style="margin-top:18px">保存设置</button></form></section>'+
      '<section class="panel"><div class="panel-header"><div><h2 class="panel-title">手动清理历史记录</h2>'+
      '<p class="panel-sub">可先查看符合条件的记录数量，再确认永久删除。</p></div></div>'+
      '<div class="tips"><div class="cleanup-fields">'+
      '<div class="field"><label for="cleanup-kind">记录类别</label><select id="cleanup-kind">'+kindOptions+'</select></div>'+
      '<div class="field"><label for="cleanup-mode">清理方式</label><select id="cleanup-mode">'+modeOptions+'</select></div>'+
      '<div class="field"><label for="cleanup-days">指定保留天数</label>'+
      '<input id="cleanup-days" type="number" min="1" max="3650" step="1" value="'+esc(request.days)+'">'+
      '<small>选择“删除早于指定天数的记录”时使用。</small></div></div>'+
      '<div class="cleanup-actions">'+
      '<button class="btn" type="button" data-action="log-cleanup-preview">查看符合条件的记录</button>'+
      '<button class="btn btn-danger" type="button" data-action="log-cleanup-run" '+(preview?.total>0?'':'disabled')+'>确认清理</button>'+
      '</div><p>结束连接记录仅清理已完成或已失效的请求。正在发送的消息及等待重试的任务不会删除。</p>'+
      '<p>服务器运行日志按日期或文件大小归档，清理时仅删除归档文件，当前日志文件继续正常写入。</p>'+
      previewHTML+'</div></section></div>';
  }
  function readLogCleanupInput(){
    const kind=document.getElementById("cleanup-kind")?.value||"";
    const mode=document.getElementById("cleanup-mode")?.value||"";
    const days=Number(document.getElementById("cleanup-days")?.value);
    if(!cleanupKinds.some(([value])=>value===kind)||!["retention","older","all"].includes(mode)){
      throw Error("请选择有效的日志类别和清理方式");
    }
    if(mode==="older"&&(!Number.isInteger(days)||days<1||days>3650)){
      throw Error("指定的保留天数必须在 1 至 3650 之间");
    }
    return {kind,mode,days:mode==="older"?days:0};
  }
  async function exportAudit(kind, filters={}){
    try{
      const params=new URLSearchParams({kind,...filters});
      const result=await fetch(API+"/audit/export?"+params.toString(),{
        method:"GET",headers:{"api-token":state.token,"Accept":"text/csv"},
        cache:"no-store",credentials:"same-origin"
      });
      if(!result.ok)throw Error("导出失败，HTTP "+result.status);
      const blob=await result.blob(),url=URL.createObjectURL(blob);
      try{
        const link=document.createElement("a");
        link.href=url;link.download=kind+".csv";
        document.body.append(link);link.click();link.remove();
      }finally{URL.revokeObjectURL(url);}
      toast("已生成 CSV 文件");
    }catch(err){toast(err.message,true);}
  }
  function clientBuilderPage(){
    const current=state.clientEditing||{},profiles=state.clientProfiles||[],tasks=state.clientTasks||[];
    const knownVersions=Array.isArray(state.clientVersions)?state.clientVersions:[];
    const platform=current.platform||"windows-x64";
    const suggested=knownVersions.find(item=>Array.isArray(item.platforms)&&item.platforms.includes(platform));
    const selectedVersion=current.version||(suggested?.version||"");
    const versionField='<div class="field"><label for="client-version">RustDesk 官方版本</label>'+
      '<input type="text" id="client-version" name="version" required maxlength="30" '+
      'pattern="[0-9]{1,3}[.][0-9]{1,3}[.][0-9]{1,4}" '+
      'list="client-version-options" placeholder="例如 1.5.3" value="'+esc(selectedVersion)+'">'+
      '<datalist id="client-version-options">'+knownVersions.map(item=>
        '<option value="'+esc(item.version)+'">'+esc(item.version)+'</option>').join("")+
      '</datalist><small>可以选择最近发布的官方版本，也可以手动填写较早版本。系统会检查所选版本是否提供对应平台的安装文件。</small></div>';
    const textField=(name,title,required=false)=>
      '<div class="field"><label for="client-'+name+'">'+title+'</label>'+
      '<input id="client-'+name+'" name="'+name+'" value="'+esc(current[name]||"")+'" '+
      (required?"required":"")+' autocomplete="off"></div>';
    const fields='<form id="client-profile-form" class="tips">'+
      '<input type="hidden" name="id" value="'+esc(current.id||"")+'">'+
      '<div class="form-grid">'+textField("name","配置名称",true)+
      '<div class="field"><label for="client-platform">客户端平台</label><select name="platform" id="client-platform">'+
      [["windows-x64","Windows x64"],["windows-arm64","Windows ARM64"],
       ["linux-x64","Linux x64（Debian/Ubuntu）"],["linux-arm64","Linux ARM64（Debian/Ubuntu）"],
       ["macos-x64","macOS x64"],["macos-arm64","macOS ARM64"]].map(([id,label])=>
        '<option value="'+id+'" '+((current.platform||"windows-x64")===id?"selected":"")+'>'+label+'</option>').join("")+
      '</select></div>'+versionField+
      textField("host","RustDesk ID 服务器地址（可带端口）",true)+
      textField("public_key","服务器公钥（Base64，32 字节）",true)+
      textField("api_server","API 地址（可留空）")+
      textField("relay_server","中继服务器地址（可留空）")+
      '</div><div class="head-actions"><button class="btn btn-primary" type="button" data-action="client-save">保存配置</button>'+
      '<button class="btn" type="button" data-action="client-new">新建配置</button></div></form>';
    const profileList=profiles.map(profile=>'<div class="status-row client-profile-row"><span><strong>'+esc(profile.name)+'</strong>'+
      '<span class="cell-sub">'+esc(profile.platform)+' · RustDesk '+esc(profile.version)+' · '+esc(profile.host)+'</span></span>'+
      '<span class="head-actions">'+
      '<button class="btn btn-sm" data-action="client-edit" data-id="'+esc(profile.id)+'">编辑</button>'+
      '<button class="btn btn-sm btn-primary" data-action="client-build" data-id="'+esc(profile.id)+'">创建安装包任务</button>'+
      '<button class="btn btn-sm btn-danger" data-action="client-delete" data-id="'+esc(profile.id)+'">删除配置</button>'+
      '</span></div>').join("")||'<p class="tips">暂无客户端配置，可以填写上方表单创建。</p>';
    const status={pending:"等待处理",building:"正在获取官方安装包",ready:"安装包已准备好",failed:"处理失败"};
    const history=tasks.map(task=>'<div class="status-row client-task-row"><span><strong>任务 #'+esc(task.id)+'</strong>'+
      '<span class="cell-sub">'+esc(task.platform)+' · RustDesk '+esc(task.version)+' · '+esc(status[task.status]||task.status)+
      (task.error?' · '+esc(task.error):'')+'</span></span>'+
      (task.status==="ready"?'<button class="btn btn-sm" data-action="client-download" data-id="'+esc(task.id)+'">下载安装程序</button>':
       task.status==="failed"?'<button class="btn btn-sm" data-action="client-retry" data-id="'+esc(task.id)+'">重新处理</button>':
       '<span class="pill muted">'+esc(status[task.status]||"处理中")+'</span>')+
      '</div>').join("")||'<p class="tips">尚未创建安装包任务。</p>';
    return pageHeading("客户端安装包构建",
      "可选择 RustDesk 已发布的官方版本。Windows 使用包含服务器配置的安装文件名，Linux 和 macOS 提供原始安装程序与配置脚本。",
      '<button type="button" class="btn" data-action="refresh">刷新任务状态</button>')+
      '<section class="panel"><div class="panel-header"><h2 class="panel-title">客户端预设配置</h2></div>'+fields+'</section>'+
      '<section class="panel" style="margin-top:16px"><div class="panel-header"><h2 class="panel-title">使用本地安装文件生成</h2></div>'+
      '<div class="tips"><p>直接选择本机 RustDesk 安装文件，浏览器会生成带服务器配置的文件。安装文件不会上传到服务器，也不会计算 SHA-256。</p>'+
      '<div class="field"><label for="client-local-file">本地安装文件（Windows EXE、Linux DEB 或 macOS DMG）</label>'+
      '<input type="file" id="client-local-file" accept=".exe,.deb,.dmg" aria-label="选择本地安装文件">'+
      '<small>在上方填写服务器地址、公钥和目标平台。使用本地文件时无须选择官方版本或保存配置，原始安装文件内容保持不变。</small></div>'+
      '<div class="head-actions"><button type="button" class="btn btn-primary" data-action="client-build-local">在浏览器生成并下载</button></div></div></section>'+
      '<section class="panel" style="margin-top:16px"><div class="panel-header"><h2 class="panel-title">已保存的配置</h2></div>'+
      '<div class="tips">'+profileList+'</div></section>'+
      '<section class="panel" style="margin-top:16px"><div class="panel-header"><h2 class="panel-title">历史构建任务及安装包</h2></div>'+
      '<div class="tips">'+history+'</div></section>'+
      '<p class="subtitle" style="margin-top:12px">安装文件来自 RustDesk 官方发行版。对官方公布 SHA-256 的版本核对摘要；未公布摘要的旧版本会在下载时计算并保存 SHA-256。Linux 和 macOS 压缩包内包含原始安装文件与配置脚本，请在目标系统检查安装和连接。</p>';
  }
  function notificationsPage(){
    const p=pages.notifications, settings=state.notifications?.settings||{};
    const configured=new Set(String(settings.events||"").split(","));
    const options=[
      ["login.failure","登录失败"],["deployment.changed","设备登记状态变更"],
      ["session.disconnect_requested","结束远程连接请求"],["session.ended","远程连接已结束"],
      ["audit.file","文件传输"]
    ];
    const checkboxes=options.map(([value,label])=>
      '<label class="check"><input type="checkbox" name="event" value="'+esc(value)+'" '+
      (configured.has(value)?"checked":"")+'>'+esc(label)+'</label>').join("");
    const body='<form id="notification-form" class="tips">'+
      '<label class="check"><input type="checkbox" name="enabled" '+
      (settings.enabled?"checked":"")+'>启用邮件事件通知</label>'+
      '<div class="field"><label for="notification-recipients">通知收件邮箱</label>'+
      '<textarea id="notification-recipients" name="recipients" rows="3" placeholder="一行一个邮箱地址，最多 5 位">'+
      esc(String(settings.recipients||"").replaceAll(",","\n"))+'</textarea>'+
      '<small>先在 SMTP 邮件服务页面启用邮件发送。收件邮箱最多填写五个。</small></div>'+
      '<div class="field"><label>需要通知的事件</label><div class="option-grid">'+checkboxes+'</div></div>'+
      '<button class="btn btn-primary" type="submit">保存通知设置</button></form>';
    const config=canPageAction("notifications","edit")?
      '<section class="panel">'+body+'</section>':
      '<section class="panel"><p class="tips">您拥有查看通知发送记录的权限。</p></section>';
    return pageHeading("邮件事件通知","选择需要接收邮件的事件，检查已发送邮件和失败原因。")+
      config+'<section class="panel" style="margin-top:16px">'+
      '<div class="panel-header"><h2 class="panel-title">发送记录</h2><button class="btn btn-sm" data-action="refresh">刷新</button></div>'+
      table(p,state.rows)+pagination()+'</section>';
  }
  function mailPage(){
    const m=state.mail||{};
    const security=m.security||"starttls";
    const options=[["starttls","STARTTLS（通常为 587 端口）"],["tls","TLS 加密连接（通常为 465 端口）"],["none","不加密（适用于可信的内部邮件服务）"]]
      .map(([value,label])=>'<option value="'+value+'" '+(value===security?"selected":"")+'>'+label+'</option>').join("");
    const textField=(name,label,value,type="text")=>
      '<div class="field"><label for="mail-'+name+'">'+esc(label)+'</label><input id="mail-'+name+
      '" type="'+type+'" name="'+name+'" value="'+esc(value||"")+'" autocomplete="off"></div>';
    const inputs=
      '<label class="field"><span>启用邮件发送</span><input name="enabled" type="checkbox" '+(m.enabled?"checked":"")+'></label>'+
      textField("host","SMTP 主机",m.host)+
      '<div class="field"><label for="mail-port">端口</label><input id="mail-port" name="port" type="number" min="1" max="65535" value="'+esc(m.port||587)+'" required></div>'+
      '<div class="field"><label for="mail-security">连接方式</label><select id="mail-security" name="security">'+options+'</select></div>'+
      textField("username","SMTP 用户名",m.username)+
      '<div class="field"><label for="mail-password">SMTP 密码</label><input id="mail-password" name="password" type="password" autocomplete="new-password" placeholder="'+(m.password_configured?"留空可保留已配置的密码":"请输入 SMTP 密码（可留空）")+'">'+
      '<small>保存密码前需在服务器设置 RUSTDESK_API_MAIL_ENCRYPTION_KEY。</small></div>'+
      '<label class="field"><span>清除已保存的 SMTP 密码</span><input name="clear_password" type="checkbox"></label>'+
      textField("sender_email","发件邮箱",m.sender_email,"email")+
      textField("sender_name","发件人名称",m.sender_name)+
      textField("base_url","用户访问控制台的公开地址",m.base_url,"url");
    return pageHeading("SMTP 邮件服务","配置邮件发送后，可以向团队成员发送测试邮件并用于账户恢复。")+
      '<div class="mail-page-grid"><section class="panel mail-settings-card"><div class="panel-header"><div><h2 class="panel-title">发送邮件设置</h2>'+
      '<p class="panel-sub">SMTP 配置保存在数据库中。密码经过加密保存，不会再次显示原文。</p></div></div>'+
      '<form id="mail-settings-form" class="tips"><div class="form-grid">'+inputs+'</div>'+
      '<button class="btn btn-primary" type="submit" style="margin-top:16px">保存邮件设置</button></form></section>'+
      '<section class="panel mail-test-card"><div class="panel-header"><div><h2 class="panel-title">发送测试邮件</h2><p class="panel-sub">先保存 SMTP 配置，再向指定邮箱发送测试邮件。</p></div></div>'+
      '<form id="mail-test-form" class="tips"><div class="form-grid">'+textField("to","收件邮箱","","email")+
      '</div><button type="submit" class="btn btn-primary" style="margin-top:16px">发送测试邮件</button></form></section></div>';
  }


  function pushPage(){
    const existing=state.pushEditing||{};
    const selected=existing.kind||"messagepusher";
    const kinds=[["messagepusher","MessagePusher"],["smtp","SMTP 邮件"],["serverchan","Server酱 Turbo"]];
    const typeLabel=Object.fromEntries(kinds);
    const eventLabels=state.pushEvents.length?state.pushEvents:[
      {code:"device.online",name:"设备上线"},{code:"device.offline",name:"设备离线"},
      {code:"device.registered",name:"新设备登记"},{code:"connection.started",name:"远程连接建立"},
      {code:"connection.ended",name:"远程连接结束"},{code:"audit.file",name:"文件传输"},
      {code:"audit.alarm",name:"设备安全告警"}
    ];
    const enabledEvents=new Set(existing.events||[]);
    const textField=(name,label,value="",type="text",placeholder="")=>
      '<div class="field"><label for="push-'+name+'">'+esc(label)+'</label><input id="push-'+name+
      '" name="'+name+'" type="'+type+'" value="'+esc(value||"")+'" placeholder="'+esc(placeholder)+
      '" autocomplete="off"></div>';
    const fieldSet=
      '<div class="form-grid">'+textField("name","渠道名称",existing.name||"", "text","例如：个人 Telegram 推送")+
      '<div class="field"><label for="push-kind">推送方式</label><select id="push-kind" name="kind" '+(existing.id?'disabled':'')+'>'+
      kinds.map(([code,label])=>'<option value="'+code+'" '+(selected===code?'selected':'')+'>'+label+'</option>').join("")+
      '</select></div></div>'+
      '<label class="check"><input type="checkbox" name="enabled" '+(existing.enabled!==false?'checked':'')+'>启用此渠道</label>'+
      '<div data-push-kind="messagepusher" '+(selected!=="messagepusher"?'hidden':'')+' class="form-grid">'+
      textField("mp_base_url","MessagePusher 服务地址",selected==="messagepusher"?existing.base_url:"","url","https://push.example.com")+
      textField("mp_api_key","发送密钥", "", "password",existing.api_key_configured?"已保存，留空保持不变":"mpk_...")+
      '</div>'+
      '<div data-push-kind="serverchan" '+(selected!=="serverchan"?'hidden':'')+' class="form-grid">'+
      textField("sc_base_url","Server酱 API 地址",selected==="serverchan"?existing.base_url||"https://sctapi.ftqq.com":"https://sctapi.ftqq.com","url","https://sctapi.ftqq.com")+
      textField("sc_api_key","SendKey", "", "password",existing.api_key_configured?"已保存，留空保持不变":"SCT...")+
      '</div>'+
      '<div data-push-kind="smtp" '+(selected!=="smtp"?'hidden':'')+' class="form-grid">'+
      textField("smtp_host","SMTP 主机",existing.smtp_host)+
      '<div class="field"><label for="push-smtp-port">SMTP 端口</label><input id="push-smtp-port" name="smtp_port" type="number" min="1" max="65535" value="'+esc(existing.smtp_port||587)+'"></div>'+
      '<div class="field"><label for="push-smtp-security">连接加密</label><select id="push-smtp-security" name="smtp_security">'+
      [["starttls","STARTTLS"],["tls","TLS"],["none","不加密（仅可信内网）"]].map(([code,label])=>
        '<option value="'+code+'" '+((existing.smtp_security||"starttls")===code?'selected':'')+'>'+label+'</option>').join("")+'</select></div>'+
      textField("smtp_username","SMTP 用户名",existing.smtp_username)+
      textField("smtp_password","SMTP 密码","","password",existing.password_configured?"已保存，留空保持不变":"请填写邮箱授权码")+
      textField("smtp_from","发件邮箱",existing.smtp_from,"email")+
      textField("smtp_to","收件邮箱",existing.smtp_to,"email")+
      '</div>'+
      '<div class="field"><label>订阅事件</label><div class="option-grid">'+
      eventLabels.map(e=>'<label class="check"><input type="checkbox" name="event" value="'+esc(e.code)+
        '" '+(enabledEvents.has(e.code)?'checked':'')+'>'+esc(e.name)+'</label>').join("")+
      '</div></div>'+
      '<div class="head-actions"><button type="submit" class="btn btn-primary">保存推送渠道</button>'+
      '<button class="btn" type="button" data-action="push-new">新建渠道</button></div>';
    const config='<section class="panel"><div class="panel-header"><div><h2 class="panel-title">'+
      (existing.id?'编辑推送渠道':'添加推送渠道')+'</h2><p class="panel-sub">各渠道独立配置，并选择需要接收的设备事件。</p></div></div>'+
      '<form id="push-channel-form" class="tips" style="display:grid;gap:16px">'+
      '<input type="hidden" name="id" value="'+esc(existing.id||"")+'">'+fieldSet+'</form></section>';
    const channels=(state.pushChannels||[]).map(x=>
      '<div class="status-row"><span><strong>'+esc(x.name)+'</strong>'+
      '<span class="cell-sub">'+esc(typeLabel[x.kind]||x.kind)+' · '+(x.enabled?'已启用':'已停用')+
      ' · 已订阅 '+esc((x.events||[]).length)+' 种事件'+
      (x.kind==="smtp"?' · '+esc(x.smtp_to||""):'')+'</span></span>'+
      '<span class="head-actions"><button class="btn btn-sm" data-action="push-edit" data-id="'+esc(x.id)+'">编辑</button>'+
      '<button class="btn btn-sm" data-action="push-test" data-id="'+esc(x.id)+'" '+(x.enabled?'':'disabled')+'>测试</button>'+
      '<button class="btn btn-sm btn-danger" data-action="push-delete" data-id="'+esc(x.id)+'">删除</button></span></div>'
    ).join("")||'<p>尚未添加推送渠道。</p>';
    const statusNames={pending:"等待发送",sending:"正在发送",retry:"等待重试",paused:"暂停发送",sent:"发送成功",failed:"发送失败"};
    const history=(state.pushHistory||[]).map(x=>
      '<div class="status-row"><span><strong>'+esc(x.title)+'</strong>'+
      '<span class="cell-sub">'+esc(statusNames[x.status]||x.status)+' · '+esc(x.event_type)+
      ' · 尝试 '+esc(x.attempts)+' 次 · '+date(x.created_at)+
      (x.last_error?' · '+esc(x.last_error):'')+'</span></span>'+
      (x.status==="failed"?'<button class="btn btn-sm" data-action="push-retry" data-id="'+esc(x.id)+'">重新发送</button>':'')+
      '</div>').join("")||'<p>目前没有发送记录。</p>';
    return pageHeading("消息推送","支持 MessagePusher、SMTP 邮件和 Server酱。通知标题优先显示事件和设备，正文包含连接双方、IP、时间及其他可用信息。")+
      '<section class="panel"><div class="panel-header"><h2 class="panel-title">已配置的渠道</h2>'+
      '<button class="btn btn-sm" data-action="refresh">刷新</button></div><div class="tips">'+channels+'</div></section>'+
      '<div style="margin-top:16px">'+config+'</div>'+
      '<section class="panel" style="margin-top:16px"><div class="panel-header"><div><h2 class="panel-title">发送记录</h2>'+
      '<p class="panel-sub">最近 20 条，共 '+esc(state.pushTotal)+' 条。发送失败时可重试。</p></div></div>'+
      '<div class="tips">'+history+'</div></section>';
  }

  function profile(){
    const m=state.mfa||{};
    const mfaText=m.enabled?
      "已启用身份验证器双重认证，剩余恢复码："+esc(m.remaining_recovery_codes??0)+" 个。":
      "尚未启用双重认证，开启后登录需要输入动态验证码。";
    const mfaActions=m.enabled?
      '<button type="button" class="btn" data-action="mfa-regenerate">重新生成恢复码</button>'+
      '<button type="button" class="btn btn-danger" data-action="mfa-disable">解除双重认证</button>':
      '<button type="button" class="btn btn-primary" data-action="mfa-setup">启用 TOTP 双重认证</button>';
    const email=state.emailVerification||{};
    const emailDescription=email.email?
      "当前邮箱："+esc(email.email)+(email.enabled?"（已启用登录验证）":"（尚未启用邮箱登录验证）"):
      "当前账户没有有效的邮箱地址，请联系管理员设置邮箱。";
    const emailButton=email.enabled?
      '<button type="button" class="btn btn-danger" data-action="email-verification-disable">关闭邮箱登录验证</button>':
      '<button type="button" class="btn btn-primary" data-action="email-verification-setup" '+
      (!email.mail_configured||!email.email?'disabled':'')+'>发送邮箱验证码并启用</button>';
    const account=state.user||{};
    const passkeys=Array.isArray(state.passkeys)?state.passkeys:[];
    const passkeyRows=passkeys.length?passkeys.map(item=>
      '<div class="passkey-item"><div class="passkey-item-info"><strong>'+esc(item.name||"Passkey")+'</strong>'+
      '<span>添加于 '+date(item.created_at)+(item.last_used_at?' · 最近使用：'+date(item.last_used_at):" · 尚未使用")+'</span></div>'+
      '<button type="button" class="btn btn-sm btn-danger" data-action="passkey-revoke" data-id="'+esc(item.id)+'">移除</button></div>').join(""):
      '<p class="passkey-empty">您还没有添加 Passkey。</p>';
    const passkeyCard='<section class="panel profile-card"><div class="panel-header"><div><h2 class="panel-title">Passkey 通行密钥</h2>'+
      '<p class="panel-sub">使用指纹、面容或屏幕锁验证后登录。</p></div></div>'+
      '<div class="profile-card-body"><p>Passkey 与当前访问域名绑定。添加后可直接使用 Passkey 登录。</p>'+
      '<div class="passkey-register-row"><div class="field"><label for="passkey-name">Passkey 名称</label>'+
      '<input id="passkey-name" maxlength="80" placeholder="例如：工作电脑或手机" value="我的设备"></div>'+
      '<button class="btn btn-primary" type="button" data-action="passkey-register">添加 Passkey</button></div>'+
      (state.passkeysError?'<p class="warning">'+esc(state.passkeysError)+'</p>':"")+
      '<div class="passkey-list">'+passkeyRows+'</div></div></section>';
    const securityCards=passkeyCard+
      '<section class="panel profile-card"><div class="panel-header"><div><h2 class="panel-title">修改登录密码</h2><p class="panel-sub">更新后其他设备需要重新登录。</p></div></div>'+
      '<form id="password-form" class="tips profile-card-body"><div class="field"><label>当前密码</label><input type="password" name="old_password" autocomplete="current-password" required></div><div class="field"><label>新密码</label><input type="password" name="new_password" autocomplete="new-password" minlength="8" maxlength="72" required></div><button class="btn btn-primary" type="submit">保存新密码</button></form></section>'+
      '<section class="panel profile-card"><div class="panel-header"><div><h2 class="panel-title">双重认证（TOTP）</h2><p class="panel-sub">'+mfaText+'</p></div></div><div class="tips profile-card-body">'+mfaActions+'</div></section>'+
      '<section class="panel profile-card"><div class="panel-header"><div><h2 class="panel-title">邮箱登录验证</h2><p class="panel-sub">'+emailDescription+'</p></div></div>'+
      '<div class="tips profile-card-body"><p>启用后通过邮件接收登录验证码。已启用 TOTP 时优先使用身份验证器。</p>'+
      emailButton+(!email.mail_configured?'<p>管理员尚未启用 SMTP 邮件服务。</p>':'')+'</div></section>';
    return pageHeading("个人设置","查看个人资料并管理账户登录与身份认证方式。")+
      '<section class="panel profile-summary"><div class="profile-summary-id"><span class="profile-avatar">'+esc((account.nickname||account.username||"U").slice(0,1)).toUpperCase()+'</span>'+
      '<div><h2>'+esc(account.nickname||account.username||"当前账户")+'</h2><p>@'+esc(account.username||"")+'</p></div></div>'+
      '<div class="profile-summary-meta"><span>电子邮箱 <strong>'+esc(account.email||"未设置")+'</strong></span><span>身份 <strong>'+(state.isAdmin?"系统管理员":"普通用户")+'</strong></span></div></section>'+
      '<div class="profile-grid">'+securityCards+'</div>';
  }
  function riskNotice(key){
      if(key==="jwt.key"||key==="app.internal-key"){
        return "更换通信密钥前，请核对 API 与 RustDesk Server 的对应设置，并安排客户端重新登录。";
      }
      if(key==="runtime.mail-encryption-key"||key==="runtime.push-encryption-key"){
        return "更换加密密钥前，请备份现有密钥与数据库，并确认已有加密凭据可以重新加密。";
      }
      if(key==="gorm.type"||key.startsWith("mysql.")||key.startsWith("postgresql.")){
        return "切换数据库前，请在新数据库恢复原有业务数据。保存时将检查目标数据库连接和迁移版本。";
      }
      if(key==="gin.api-addr"||key==="gin.trust-proxy"){
        return "修改监听地址或可信代理前，请确认管理控制台的网络访问方式。";
      }
      if(key==="runtime.push-allow-private-urls"){
        return "此设置允许推送服务访问内部网络，启用前应确认相关网络访问权限。";
      }
      return "";
  }

  function settingsPage(){
    const current=state.settings||{items:[]},items=current.items||[];
    const filtered=items.filter(item=>
      (state.settingsCategory==="all"||item.category===state.settingsCategory)&&
      (!state.settingsSearch||[item.key,item.name,item.description,item.environment]
        .join(" ").toLowerCase().includes(state.settingsSearch.toLowerCase())));
    const categories=[...new Set(items.map(item=>item.category))].sort();
    const actions=[
      ["mail","SMTP 邮件服务"],["notifications","邮件通知"],["retention","日志保留期限"],
      ["policies","设备策略"],["controlRoles","控制角色"],["relayNodes","中继节点"],
      ["accessRules","连接访问规则"],["clientBuilder","安装包任务"],["push","个人推送"],
      ["oauth","OIDC / SSO"],["adminRoles","管理员权限角色"],["profile","个人双重认证"]
    ];
    const summary=state.settingsOverview||{},outbox=summary.outbox||{};
    const specialized=summary.summary||{};
    const mailInfo=specialized.smtp_settings||{};
    const noticeInfo=specialized.notification_settings||{};
    const retainInfo=specialized.audit_retention||{};
    const specializedDetails='<div class="settings-highlights">'+
      '<div class="settings-highlight"><h3>SMTP 邮件服务</h3>'+
      '<p>SMTP：'+esc(mailInfo.enabled?"已启用":"未启用")+'</p>'+
      '<p>服务地址：'+esc(mailInfo.host||"未配置")+(mailInfo.port?'：'+esc(mailInfo.port):"")+'</p>'+
      '<p>发件邮箱：'+esc(mailInfo.sender_email||"未配置")+'</p></div>'+
      '<div class="settings-highlight"><h3>管理员邮件通知</h3>'+
      '<p>状态：'+esc(noticeInfo.enabled?"已启用":"未启用")+'</p>'+
      '<p>事件：'+esc(noticeInfo.events||"未配置")+'</p></div>'+
      '<div class="settings-highlight"><h3>数据保留</h3>'+
      '<p>连接日志 '+esc(retainInfo.connection_days??0)+' 天；登录记录 '+
      esc(retainInfo.login_days??0)+' 天</p>'+
      '<p>0 天表示不自动清理。</p></div>'+
      '<div class="settings-highlight"><h3>策略与节点</h3>'+
      '<p>设备策略：'+esc(specialized.device_policies??0)+'；连接规则：'+
      esc(specialized.access_rules??0)+'</p>'+
      '<p>中继节点：'+esc(specialized.relay_nodes??0)+'；个人推送渠道：'+
      esc(specialized.push_channels??0)+'</p></div></div>';
    const links=actions.map(([id,label])=>'<button type="button" class="btn btn-sm settings-feature-link" data-view="'+id+'"><span>'+esc(label)+'</span>'+icon("arrow",14)+'</button>').join("");
    const itemView=item=>{
      const id="setting-value-"+item.key;
      const saved=item.override_enabled?item.stored_value:item.current_value;
      let input="";
      if(item.sensitive){
        input='<input id="'+esc(id)+'" type="password" autocomplete="new-password" placeholder="'+
          (item.override_enabled?"留空保持原来的覆盖值":"填写需要覆盖的新值")+'">';
      }else if(item.value_type==="bool"){
        const selected=String(saved)==="true"||String(saved)==="1";
        input='<select id="'+esc(id)+'"><option value="true" '+(selected?"selected":"")+
          '>启用</option><option value="false" '+(!selected?"selected":"")+
          '>停用</option></select>';
      }else{
        input='<input id="'+esc(id)+'" type="'+(item.value_type==="int"?"number":"text")+
          '" value="'+esc(saved??"")+'" autocomplete="off">';
      }
      const run=item.sensitive?(item.configured?"已配置（隐藏具体内容）":"未配置"):
        (item.current_value===""?"（空字符串）":esc(item.current_value));
      const warning=riskNotice(item.key);
      const status=item.pending_restart?
        '<span class="pill warn">待重启生效</span>':
        '<span class="pill muted">运行中</span>';
      return '<article class="panel settings-item" data-setting-item="'+esc(item.key)+'">'+
        '<div class="panel-header settings-item-head"><div><h2 class="panel-title">'+esc(item.name)+'</h2>'+
        '<p class="panel-sub"><code>'+esc(item.key)+'</code></p></div>'+status+'</div>'+
        '<div class="settings-item-body">'+
        '<p class="settings-item-description">'+esc(item.description)+'</p>'+
        (warning?'<p class="settings-item-warning" role="note">'+esc(warning)+'</p>':"")+
        '<div class="settings-current"><div><span class="settings-meta-label">当前运行值</span>'+
        '<strong class="settings-current-value">'+run+'</strong></div>'+
        '<span class="pill">'+esc(item.source)+'</span></div>'+
        '<p class="settings-item-meta">环境变量：<code>'+esc(item.environment)+'</code> · '+
        (item.bootstrap?'本机启动设置':'数据库中的应用设置')+' · 修改后重新启动 API</p>'+
        (item.active?
          '<div class="field"><label for="'+esc(id)+'">网页覆盖值</label>'+input+'</div>'+
          '<div class="settings-item-actions"><button type="button" class="btn btn-primary btn-sm" data-action="setting-save" data-id="'+esc(item.key)+'">保存并启用覆盖</button>'+
          (item.override_enabled?'<button type="button" class="btn btn-sm" data-action="setting-reset" data-id="'+esc(item.key)+'">关闭网页覆盖</button>':"")+
          '</div>':'<p class="settings-item-meta">此项尚未在当前程序中使用。</p>')+
        '</div></article>';
    };
    const show=filtered.map(itemView).join("")||'<div class="empty">没有匹配的设置项。</div>';
    const history=(state.settingsHistory||[]).map(row=>
      '<div class="status-row"><span><strong>'+esc(row.setting_key)+'</strong>'+
      '<span class="cell-sub">'+(row.enabled?"启用网页覆盖":"关闭网页覆盖")+
      ' · '+esc(row.storage==="bootstrap"?"启动设置":"应用设置")+'</span></span>'+
      '<span class="cell-sub">'+esc(date(row.changed_at))+'</span></div>').join("");
    const push=outbox.push||{},mail=outbox.notification||{};
    const failure=(state.settingsFailed||[]).map(row=>
      '<div class="status-row"><span><strong>事件 #'+esc(row.Id||row.id)+'</strong>'+
      '<span class="cell-sub">失败次数：'+esc(row.Attempts||row.attempts||0)+' · '+
      esc(row.LastError||row.last_error||"")+'</span></span>'+
      '<button class="btn btn-sm" data-action="settings-outbox-retry" data-id="'+esc(row.Id||row.id)+
      '" data-category="'+esc(state.settingsFailedCategory)+'">重试</button></div>').join("");
    return pageHeading("设置中心","查看每项设置的实际运行值、来源及用途，保存后按提示重新启动 API。")+
      '<section class="panel settings-section"><div class="panel-header"><h2 class="panel-title">专项功能</h2></div>'+
      '<div class="settings-section-body">'+
      '<p class="settings-section-intro">邮件、策略和用户推送等设置继续使用各自的管理页面。</p>'+
      '<nav class="settings-feature-links" aria-label="专项功能">'+links+'</nav>'+
      specializedDetails+'</div></section>'+
      '<section class="panel settings-section"><div class="panel-header"><h2 class="panel-title">通知任务状态</h2></div>'+
      '<div class="settings-section-body">'+
      '<div class="settings-outbox-grid">'+
      '<div class="settings-outbox-card"><h3>个人推送</h3><p><strong>'+esc(push.pending||0)+'</strong> 条等待处理，<strong>'+esc(push.failed||0)+'</strong> 条失败</p>'+
      '<p class="settings-outbox-time">最早待处理事件：'+esc(push.oldest_pending_at?date(push.oldest_pending_at):"暂无")+'</p></div>'+
      '<div class="settings-outbox-card"><h3>管理员邮件</h3><p><strong>'+esc(mail.pending||0)+'</strong> 条等待处理，<strong>'+esc(mail.failed||0)+'</strong> 条失败</p>'+
      '<p class="settings-outbox-time">最早待处理事件：'+esc(mail.oldest_pending_at?date(mail.oldest_pending_at):"暂无")+'</p></div></div>'+
      '<div class="settings-outbox-actions" role="group" aria-label="失败事件类别">'+
      '<button type="button" class="btn btn-sm '+(state.settingsFailedCategory==="push"?"btn-primary":"")+
      '" data-action="settings-outbox-category" data-category="push">个人推送的失败事件</button>'+
      '<button type="button" class="btn btn-sm '+(state.settingsFailedCategory==="notification"?"btn-primary":"")+
      '" data-action="settings-outbox-category" data-category="notification">管理员邮件的失败事件</button></div>'+
      (failure?'<div class="settings-failed-list">'+failure+'</div>':'<p class="settings-section-intro">此类别暂无处理失败的事件。</p>')+
      '</div></section>'+
      '<section class="panel settings-section"><div class="settings-section-body settings-filter-body">'+
      '<div class="form-grid">'+
      '<div class="field"><label for="settings-search">搜索设置</label><input id="settings-search" value="'+esc(state.settingsSearch)+
      '" placeholder="搜索名称、用途、配置名称或环境变量"></div>'+
      '<div class="field"><label for="settings-category">设置类别</label><select id="settings-category">'+
      '<option value="all">全部类别</option>'+categories.map(name=>'<option value="'+esc(name)+'" '+
      (name===state.settingsCategory?"selected":"")+'>'+esc(name)+'</option>').join("")+
      '</select></div></div>'+
      '<p class="settings-section-intro">共 '+items.length+' 项配置，当前显示 '+filtered.length+' 项。覆盖设置的值会保存在持久化目录或数据库中。</p>'+
      '</div></section>'+
      '<div class="settings-grid">'+show+'</div>'+
      (history?'<section class="panel settings-section settings-history"><div class="panel-header"><h2 class="panel-title">最近设置修改记录</h2></div><div class="settings-section-body">'+history+'</div></section>':"");

  }
  function renderPage(){
    if(state.page==="overview")return overview();
    if(state.page==="contacts")return contacts();
    if(state.page==="shared")return shared();
    if(state.page==="server")return server();
    if(state.page==="settings")return settingsPage();
    if(state.page==="retention")return retentionPage();
    if(state.page==="mail")return mailPage();
    if(state.page==="notifications")return notificationsPage();
    if(state.page==="clientBuilder")return clientBuilderPage();
    if(state.page==="push")return pushPage();
    if(state.page==="profile")return profile();
    return genericList();
  }
  function lookupKind(name,row){
    if(name==="id"&&state.page==="addressbooks")return "device";
    if(name==="to_id")return ({1:"user",2:"user_group",3:"all"})[String(document.querySelector('#edit-form [name="type"]')?.value||row.type||1)]||"all";
    if(name==="source_id")return ({all:"all",user:"user",group:"user_group"})[document.querySelector('#edit-form [name="source_type"]')?.value||row.source_type||"all"]||"all";
    if(name==="target_id"){
      const kind=document.querySelector('#edit-form [name="target_type"]')?.value||row.target_type||(state.page==="controlAssignments"?"device":"all");
      return ({all:"all",device:"device",user:"user",device_group:"device_group"})[kind]||"all";
    }
    if(name==="scope_id")return ({global:"all",group:"user_group",individual:"user"})[document.querySelector('#edit-form [name="scope_type"]')?.value||row.scope_type||"global"]||"all";
    if(["user_id","owner_user_id","from_user_id","to_user_id","requested_by"].includes(name))return "user";
    if(["device_id","peer_id","from_peer"].includes(name))return "device";
    if(name==="group_id"||name==="auto_group_ids")return ["devices","groupPolicyAssignments"].includes(state.page)?"device_group":"user_group";
    if(name==="collection_id")return "collection";
    if(name==="policy_id")return "policy";
    if(name==="role_id")return state.page==="adminRoleAssignments"?"admin_role":"control_role";
    return "";
  }
  const lookupLabels={user:"用户",device:"设备",user_group:"用户分组",device_group:"设备组",collection:"地址簿",policy:"安全策略",control_role:"控制角色",admin_role:"管理角色",deployment_record:"设备部署记录",trash_record:"回收站记录"};
  function pickerField(name,value,required,row,inputID=""){
    const kind=lookupKind(name,row),selected=String(value??"");
    const all=kind==="all",display=selected&&selected!=="0"?referenceText(kind,selected):"";
    const dep={to_id:"type",source_id:"source_type",target_id:"target_type",scope_id:"scope_type"}[name]||"";
    const empty=all?name==="target_id"?"":"0":selected;
    return '<div class="entity-picker" data-picker="'+esc(kind)+'" data-picker-dep="'+esc(dep)+'" data-picker-field="'+esc(name)+'">'+
      '<div class="picker-control"><button type="button" class="picker-trigger" data-action="picker-open" '+
      (all?"disabled":"")+' aria-label="选择'+esc(lookupLabels[kind]||"对象")+'">'+
      '<span class="picker-trigger-label">'+esc(all?"适用于全部对象":display||"点击选择"+(lookupLabels[kind]||"对象"))+'</span>'+icon("search",17)+'</button>'+
      '<input type="hidden" class="picker-id" name="'+esc(name)+'" id="'+esc(inputID)+'" value="'+esc(empty)+'">'+
      '<button type="button" class="btn btn-sm" data-action="picker-clear" '+(all?"disabled":"")+'>清除</button></div>'+
      '<div class="picker-results" hidden></div>'+
      (kind==="device"?'<div class="picker-manual"><input type="text" class="picker-manual-value" maxlength="253" aria-label="手动填写设备 ID" placeholder="外部设备 ID">'+
      '<button class="btn btn-sm" type="button" data-action="picker-manual-single">使用此设备 ID</button></div>':"")+
      '<small class="picker-selection">'+esc(all?"适用于全部对象":display?"当前选择："+display:"请选择需要关联的对象")+'</small></div>';
  }
  function multiPicker(id,kind="user",initial=""){
    const ids=[...new Set(String(initial||"").split(/[,;，\s]+/).map(x=>x.trim()).filter(Boolean))];
    const object=lookupLabels[kind]||"用户";
    return '<div class="entity-picker entity-picker-multi" data-picker="'+esc(kind)+'" data-target="'+esc(id)+'">'+
      '<div class="picker-control"><button type="button" class="picker-trigger" data-action="picker-open" aria-label="选择'+object+'">'+
      '<span class="picker-trigger-label">点击搜索并选择'+object+'</span>'+icon("search",17)+'</button>'+
      '<button type="button" class="btn btn-sm" data-action="picker-clear">全部清除</button></div>'+
      '<input type="hidden" id="'+esc(id)+'" name="'+esc(id)+'" value="'+esc(ids.join(","))+'">'+
      '<div class="picker-results" hidden></div>'+
      (kind==="device"?'<div class="picker-manual"><input type="text" class="picker-manual-value" aria-label="填写外部设备 ID 或匹配规则" placeholder="外部设备 ID 或 desk-* 等匹配规则" maxlength="253">'+
      '<button type="button" class="btn btn-sm" data-action="picker-manual-add">添加设备 ID</button></div>':"")+
      '<div class="picker-selected" aria-live="polite">'+
      ids.map(value=>'<button type="button" class="picker-chip" data-action="picker-remove" data-id="'+esc(value)+'" aria-label="移除'+object+' '+esc(value)+'">'+
        (kind==="device"?deviceStatusDot(value):"")+esc(["user_group","deployment_record","trash_record"].includes(kind)?referenceText(kind,value):object+" "+value)+' ×</button>').join("")+'</div>'+
      '<small class="picker-selection">'+(ids.length?"已选择 "+ids.length+" 个"+object:
        kind==="device"?"选择设备或填写外部设备 ID，留空表示适用于所有设备":"请选择"+object)+'</small></div>';
  }
  function closePickerDialog(){
    root.querySelector(".picker-overlay")?.remove();
    state.pickerDialog=null;
  }
  async function openPickerDialog(picker){
    if(!picker||picker.dataset.picker==="all")return;
    closePickerDialog();
    const kind=picker.dataset.picker,object=lookupLabels[kind]||"对象";
    const overlay='<div class="picker-overlay" role="presentation">'+
      '<section class="picker-dialog" data-picker="'+esc(kind)+'" role="dialog" aria-modal="true" aria-label="选择'+esc(object)+'">'+
      '<header class="picker-dialog-head"><div><h2>选择'+esc(object)+'</h2><p>可根据名称、编号或其他信息搜索。选择后自动填写对应编号。</p></div>'+
      '<button type="button" class="icon-btn" data-action="picker-close" aria-label="关闭搜索">'+icon("close")+'</button></header>'+
      '<div class="picker-dialog-body"><div class="picker-dialog-toolbar"><input type="search" class="picker-query" aria-label="搜索'+esc(object)+'" placeholder="输入名称、编号或关键词" autocomplete="off">'+
      '<button type="button" class="btn btn-primary" data-action="picker-search">搜索</button></div>'+
      '<div class="picker-results" role="listbox"><div class="loading-table">正在读取可选择的'+esc(object)+'…</div></div></div>'+
      '<footer class="picker-dialog-footer"><span>选择结果后将显示名称和编号</span>'+
      '<button type="button" class="btn" data-action="picker-close">完成</button></footer></section></div>';
    root.insertAdjacentHTML("beforeend",overlay);
    const dialog=root.querySelector(".picker-dialog");
    state.pickerDialog={picker,dialog};
    dialog.querySelector(".picker-query")?.focus();
    await lookupPicker(dialog);
  }
  async function lookupPicker(picker){
    if(!picker||picker.dataset.picker==="all")return;
    const kind=picker.dataset.picker,q=picker.querySelector(".picker-query")?.value.trim()||"";
    const results=picker.querySelector(".picker-results");
    if(!results)return;
    const sequence=(Number(picker.dataset.sequence)||0)+1;
    picker.dataset.sequence=String(sequence);
    results.hidden=false;
    if(!q&&kind==="user"&&!state.isAdmin&&!hasPrivilege("users.view")){results.textContent="请输入用户名、昵称或邮箱开始搜索";return;}
    results.textContent="正在搜索…";
    try{
      let rows=[];
      if(kind==="user"&&!state.isAdmin&&!hasPrivilege("users.view")){
        const data=await request("/my/directory/users?keyword="+encodeURIComponent(q));
        rows=data.list||[];
      }else{
        const paths={
          user:"/user/list",device:state.isAdmin||hasPrivilege("devices.view")?"/peer/list":"/my/directory/devices",
          user_group:state.isAdmin?"/group/list":"/my/group/list",
          device_group:"/device_group/list",collection:state.modal?.kind==="rules-batch"||!state.isAdmin?"/my/address_book_collection/list":"/address_book_collection/list",
          policy:"/device_policy/list",control_role:"/control_role/list",admin_role:"/admin_role/list",deployment_record:"/device_deployment/list",trash_record:"/my/address_book/trash/list"
        };
        const path=paths[kind];
        if(!path){results.textContent="无法查询此类对象";return;}
        const filter=(kind==="user"||kind==="device")?"&keyword="+encodeURIComponent(q):"";
        const data=await request(path+"?page=1&page_size=200"+filter);
        rows=Array.isArray(data)?data:data?.list||[];
        if(kind==="device"&&rows.length){
          const ids=[...new Set(rows.map(row=>String(row.id||"")).filter(Boolean))].slice(0,100);
          try{
            const reply=await request("/geoip/lookup","POST",{device_ids:ids});
            for(const id of ids)state.devicePresence[id]=Object.prototype.hasOwnProperty.call(reply.device_status||{},id)?reply.device_status[id]===true:null;
          }catch{}
        }
        if(!filter){
          const normalized=q.toLowerCase();
          rows=rows.filter(item=>[item.id,item.row_id,item.device_id,item.hostname,item.name,item.alias,item.username,item.nickname,item.email,item.status].some(value=>String(value??"").toLowerCase().includes(normalized)));
        }
      }
      if(!picker.isConnected||Number(picker.dataset.sequence)!==sequence)return;
      results.innerHTML=rows.length?rows.slice(0,50).map(item=>{
        const id=kind==="trash_record"?item.row_id:item.id;
        const name=kind==="device"?[item.alias,item.hostname,item.username,item.os].filter(Boolean).join(" · "):
          kind==="deployment_record"?[item.device_id,item.status==="approved"?"已批准":item.status==="revoked"?"已撤销":item.status].filter(Boolean).join(" · "):
          kind==="trash_record"?[item.alias||item.id,item.hostname||item.id].filter(Boolean).join(" · "):
          [item.name,item.nickname||item.username,item.email].filter(Boolean).join(" · ");
        if(!state.referenceNames[kind])state.referenceNames[kind]={};
        state.referenceNames[kind][String(id)]=name||String(id);
        return '<div role="option" class="picker-result-row"><button type="button" class="picker-result" data-action="picker-select" data-id="'+esc(id)+'" data-username="'+esc(item.username||"")+'" data-title="'+esc(name)+'">'+
          '<strong>'+(kind==="device"?deviceStatusDot(id):"")+esc(name||id)+'</strong><span>编号：'+esc(id)+(name?" · "+esc(item.email||item.hostname||item.os||""):"")+'</span></button>'+
          (["user","device"].includes(kind)?'<button type="button" class="picker-result-detail" data-action="entity-detail" data-entity-type="'+kind+'" data-entity-id="'+esc(id)+'" data-entity-label="'+esc(name||id)+'">查看信息</button>':"")+'</div>';
      }).join(""):'<p class="picker-empty">没有找到匹配项，请尝试其他关键词。</p>';
    }catch(err){
      if(!picker.isConnected||Number(picker.dataset.sequence)!==sequence)return;
      results.textContent=err.message;
    }
  }
  function fieldInput(field,row){
    const [name,label,type="text",required=false]=field;
    const value=row&&row[name]!==undefined?row[name]:type==="status"?1:type==="group_type"?1:type==="rule_type"?1:type==="rule"?1:
      type==="checkbox"?(name==="enabled"&&["adminRoles","accessRules"].includes(state.page)):type==="permission_state"?0:type==="control_target"?"device":type==="admin_scope"?"global":type==="access_source"?"all":type==="access_target"?"all":type==="access_effect"?"allow":"";
    let input;
    if(type==="checkbox")input='<label class="check"><input type="checkbox" name="'+esc(name)+'" '+(value?"checked":"")+'>启用</label>';
    else if(type==="admin_permissions"){
      const enabled=new Set(String(value||"").split(",").map(x=>x.trim()));
      input='<div class="option-grid" style="display:grid;grid-template-columns:repeat(auto-fill,minmax(185px,1fr));gap:8px">'+
        administratorPermissions.map(([permission,title])=>
          '<label class="check"><input type="checkbox" name="permissions" value="'+esc(permission)+'" '+
          (enabled.has(permission)?"checked":"")+'>'+esc(title)+'</label>').join("")+'</div>';
    }
    else if(["policy_access","policy_bool","policy_approve"].includes(type)){
      const opts={
        policy_access:[["","沿用设备设置"],["custom","自定义权限"],["full","允许完整控制"],["view","仅查看"]],
        policy_bool:[["","沿用设备设置"],["Y","允许"],["N","禁止"]],
        policy_approve:[["","沿用设备设置"],["password-click","密码或手动确认"],["password","仅密码"],["click","仅手动确认"]]
      }[type];
      input='<select name="'+esc(name)+'">'+opts.map(([key,label])=>
        '<option value="'+esc(key)+'" '+(String(value)===String(key)?"selected":"")+'>'+
        esc(label)+'</option>').join("")+'</select>';
    }
    else if(name==="id_whitelist")input=multiPicker(name,"device",value);
    else if(name==="auto_group_ids")input=multiPicker(name,"user_group",value);
    else if(type==="textarea")input='<textarea name="'+esc(name)+'" rows="3">'+esc(value)+'</textarea>';
    else if(type==="color"){
      const color=Number(value===0||value==="0"?0:value||0xff5b9bd5)>>>0;
      const rgb="#"+(color&0xffffff).toString(16).padStart(6,"0");
      const alpha=color>>>24;
      input='<div class="color-picker"><input name="'+esc(name)+'" type="color" value="'+rgb+'" required aria-label="选择颜色">'+
        '<label>透明度（0 表示完全透明，255 表示完全不透明）<input type="range" name="'+esc(name)+'_alpha" min="0" max="255" step="1" value="'+alpha+'"></label></div>';
    }
    else if(lookupKind(name,row))input=pickerField(name,value,required,row);
    else if(type==="trusted_days"){
      const choices=[[0,"不限期限"],[1,"1 天"],[7,"7 天"],[30,"30 天"],[90,"90 天"]];
      input='<select name="'+esc(name)+'">'+choices.map(([days,text])=>'<option value="'+days+'" '+(Number(value)===days?"selected":"")+'>'+text+'</option>').join("")+'</select>';
    }
    else if(["group_type","status","oauth_type","rule_type","rule","permission_state","control_target","admin_scope","access_source","access_target","access_effect"].includes(type)){
      const opts={group_type:[[1,"普通组"],[2,"共享组"],[3,"公共锚点组"]],status:[[1,"已启用"],[2,"已停用"]],oauth_type:[["oidc","OIDC"],["github","GitHub"],["google","Google"],["linuxdo","Linux.do"]],rule_type:[[1,"用户"],[2,"群组"],[3,"所有用户"]],rule:[[1,"只读"],[2,"读写"],[3,"完全控制"]],permission_state:[[0,"沿用设备设置"],[1,"禁止"],[2,"允许"]],control_target:[["device","单台设备"],["user","设备所有者"],["device_group","设备组"]],admin_scope:[["global","全部账户"],["group","指定用户组"],["individual","指定用户"]],access_source:[["all","所有已登录用户"],["user","指定用户"],["group","指定用户组"]],access_target:[["all","所有设备"],["device","指定设备"],["user","指定所有者的设备"],["device_group","指定设备组"]],access_effect:[["allow","允许访问"],["deny","禁止访问"]]}[type];
      input='<select name="'+esc(name)+'">'+opts.map(o=>'<option value="'+esc(o[0])+'" '+(String(o[0])===String(value)?"selected":"")+'>'+esc(o[1])+'</option>').join("")+'</select>';
    }else input='<input name="'+esc(name)+'" type="'+esc(type)+'" value="'+esc(value)+'" '+(required?'required':"")+' autocomplete="off">';
    return '<div class="field '+(type==="textarea"?"full":"")+'"><label>'+esc(label)+(required?' <span class="required">*</span>':"")+'</label>'+input+(name==="auto_group_ids"?'<small>例如：2,3。登录后将自动加入这些组，并保留原有主要分组。</small>':"")+'</div>';
  }
  function entityDataRow(label,value,html=false){
    if(value===undefined||value===null||String(value)==="")return "";
    return '<div class="entity-data-row"><dt>'+esc(label)+'</dt><dd>'+(html?value:esc(value))+'</dd></div>';
  }
  function entityPanel(m){
    const info=m.info||{},device=m.entityType==="device";
    const id=String(m.entityId);
    const status=state.devicePresence[id];
    const sections=device?[
      ["设备 ID",id],["在线状态",status===true?"在线":status===false?"离线":"状态暂不可用"],
      ["备注",info.alias],["主机名",info.hostname],["操作系统",info.os],
      ["操作系统用户",info.username],["最后在线时间",info.last_online_time?date(info.last_online_time):"",true],
      ["最近上线 IP",info.last_online_ip?showIP(info.last_online_ip):"",true],
      ["设备备注",info.note],
      ["所有者",info.owner_username?entityLink("user",info.user_id||info.owner_username,info.owner_username):"",true],
      ["所有者用户 ID",info.user_id]
    ]:[
      ["用户编号",info.id||(/^[0-9]+$/.test(id)?id:"")],
      ["用户名",info.username||(/^[0-9]+$/.test(id)?"":id)],
      ["昵称",info.nickname],["邮箱",info.email],
      ["主要分组",referenceText("user_group",info.group_id)],["备注",info.remark]
    ];
    const rows=sections.map(([label,value,html])=>entityDataRow(label,value,html)).join("");
    const owned=Boolean(info.owned||state.page==="mydevices"&&state.rows.some(row=>String(row.id)===id)||info.user_id&&Number(info.user_id)===Number(state.user?.id));
    const editable=device?
      !m.found?"":owned?"mydevices":available("devices")?"devices":available("shared")?"shared":"":
      available("users")?"users":(String(state.user?.id)===id||String(state.user?.username)===id)?"profile":"";
    m.destination=editable;
    const summary='<div class="entity-summary">'+(device?'<div class="entity-presence">'+deviceStatusDot(id)+'<strong>'+esc(m.label||id)+'</strong></div>':'<strong>'+esc(m.label||id)+'</strong>')+
      (m.loading?'<p class="subtitle">正在读取可查看的详情…</p>':"")+
      '<dl class="entity-data">'+rows+'</dl>'+
      (m.loading?"":!m.found?'<p class="subtitle">当前账户可查看的信息已显示。其他详情需要相应管理权限。</p>':"")+'</div>';
    const connectButton=device?'<button type="button" class="btn btn-primary" data-action="device-connect" data-device-id="'+esc(id)+'" data-device-name="'+esc(info.alias||info.hostname||(m.label!==id?m.label:""))+'">连接设备</button>':"";
    const buttons=editable?'<button type="button" class="btn" data-action="entity-open-list">'+
      esc(editable==="devices"?"在设备管理中查看":editable==="mydevices"?"查看设备":editable==="shared"?"在共享设备中查看":editable==="users"?"在用户管理中查看":"打开个人设置")+'</button>':"";
    const adminUser=!device&&Number(info.id)>0&&available("users");
    const userActions=adminUser?
      (canPageAction("users","edit")?'<button type="button" class="btn btn-primary" data-action="entity-admin-edit" data-id="'+esc(info.id)+'">编辑用户资料</button>':"")+
      (hasPrivilege("users.edit_password")?'<button type="button" class="btn" data-action="reset-password" data-id="'+esc(info.id)+'">重设密码</button>':"")+
      (hasPrivilege("users.manage_2fa")?'<button type="button" class="btn btn-danger" data-action="reset-user-mfa" data-id="'+esc(info.id)+'">清除双重认证</button>':""):"";
    return modal(device?"设备信息":"用户信息",device?"设备 ID："+id:"用户："+(m.label||id),summary,
      '<button type="button" class="btn" data-action="entity-back">'+(m.previous?"返回":"关闭")+'</button>'+connectButton+userActions+buttons);
  }
  async function openEntityDetail(type,id,label){
    if(!["device","user"].includes(type)||!String(id||"").trim())return;
    const m={kind:"entity",entityType:type,entityId:String(id),label:label||String(id),
      previous:state.modal,loading:true,found:false,info:{}};
    state.modal=m;render();
    try{
      let data=null;
      if(type==="device"){
        data=state.rows.find(row=>deviceRowPages.has(state.page)&&String(row.id)===String(id))||
          state.metrics?.recent?.find(row=>String(row.id)===String(id))||null;
        try{
          const detail=await request("/my/directory/device?id="+encodeURIComponent(id));
          if(detail?.id)data=detail;
        }catch{
          if(!data&&available("devices")){
            const list=await request("/peer/list?page=1&page_size=80&keyword="+encodeURIComponent(id));
            data=(list.list||[]).find(row=>String(row.id)===String(id))||null;
          }
        }
        if(state.devicePresence[id]===undefined){
          try{
            const presence=await request("/geoip/lookup","POST",{device_ids:[String(id)]});
            state.devicePresence[id]=Object.prototype.hasOwnProperty.call(presence.device_status||{},id)?presence.device_status[id]===true:null;
          }catch{state.devicePresence[id]=null;}
        }
      }else{
        data=state.page==="users"?state.rows.find(row=>String(row.id)===String(id)||row.username===String(id)):null;
        if(!data&&(state.isAdmin||hasPrivilege("users.view"))){
          const list=await request("/user/list?page=1&page_size=80&keyword="+encodeURIComponent(id));
          data=(list.list||[]).find(row=>String(row.id)===String(id)||row.username===String(id))||null;
        }
        if(!data){
          try{data=await request("/my/directory/user?id="+encodeURIComponent(id));}catch{}
        }
        if(!data&&(String(state.user?.id)===String(id)||String(state.user?.username)===String(id)))data=state.user;
      }
      if(state.modal!==m)return;
      m.info=data||{};m.found=!!data;
    }catch{
      if(state.modal!==m)return;
    }
    m.loading=false;render();
  }
  function modal(title,desc,body,footer){
    return '<div class="overlay" id="modal-overlay"><section class="drawer" role="dialog" aria-modal="true" aria-label="'+esc(title)+'"><div class="drawer-head"><div><h2>'+esc(title)+'</h2><p>'+esc(desc||"")+'</p></div><button type="button" class="icon-btn" aria-label="关闭" data-action="close-modal">'+icon("close")+'</button></div><div class="drawer-body">'+body+'</div><div class="drawer-footer">'+footer+'</div></section></div>';
  }
  function renderModal(){
    const m=state.modal;if(!m)return "";
    if(m.kind==="entity")return entityPanel(m);
    if(m.kind==="edit"){
      const p=pages[activeKey()],fields=(p.fields||[]).filter(f=>{
        if(state.page==="users"&&m.action==="edit"&&f[0]==="password")return false;
        if(!state.isAdmin&&state.page==="addressbooks"&&(f[0]==="user_id"))return false;
        return true;
      }).map(f=>state.page==="oauth"&&m.action==="edit"&&f[0]==="client_secret"?[f[0],"Client Secret（留空表示保持当前密钥）",f[2],false]:f);
      const body='<form id="edit-form"><input type="hidden" name="id" value="'+esc(m.data.id||"")+'"><input type="hidden" name="row_id" value="'+esc(m.data.row_id||"")+'"><div class="form-grid">'+fields.map(f=>fieldInput(f,m.data)).join("")+'</div></form>';
      const saveButton=state.page==="apiTokens"?
        '<button class="btn btn-primary" type="button" data-action="submit-api-token">创建令牌</button>':
        state.page==="invitations"?
        '<button class="btn btn-primary" type="button" data-action="submit-invitation">发送邀请</button>':
        state.page==="controlRoles"?
        '<button class="btn btn-primary" type="button" data-action="submit-control-role">保存控制角色</button>':
        state.page==="adminRoles"?
        '<button class="btn btn-primary" type="button" data-action="submit-admin-role">保存管理员角色</button>':
        state.page==="accessRules"?
        '<button class="btn btn-primary" type="button" data-action="submit-access-rule">保存连接规则</button>':
        '<button class="btn btn-primary" type="submit" form="edit-form">保存</button>';
      return modal(m.action==="create"?"新建"+p.name:"编辑"+p.name,"修改完成后保存，变更会立即生效。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+saveButton);
    }
    if(m.kind==="csv"){
      const importing=m.operation==="import";
      const ownerInput=state.isAdmin?'<div class="field"><label>所属账户</label>'+
        pickerField("user_id",state.user?.id||1,true,{},"csv-owner")+
        '<small>管理员可以指定需要管理的用户账户。</small></div>':"";
      const upload=importing?'<div class="field"><label for="csv-file">选择 UTF-8 CSV 文件</label>'+
        '<input id="csv-file" type="file" accept=".csv,text/csv" required>'+
        '<small>最多 4 MB、5000 条设备。根据设备 ID 和地址簿编号更新同一条目，保留原有连接密码。</small></div>':
        '<p>将导出设备 ID、别名、主机名、系统、标签、备注等字段。连接密码和认证信息不包含在导出文件中。</p>';
      const body='<div class="form-grid">'+ownerInput+upload+'</div>';
      return modal(importing?"导入地址簿 CSV":"导出地址簿 CSV","仅处理指定账户的地址簿条目。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" data-action="csv-submit">'+(importing?"导入 CSV":"下载 CSV")+'</button>');
    }
    if(m.kind==="email-verification"){
      const body='<p>验证码已发送到当前账户邮箱，有效期为 10 分钟。</p>'+
        '<form id="email-verification-form"><div class="field">'+
        '<label for="email-verification-code">6 位邮箱验证码</label>'+
        '<input id="email-verification-code" name="code" inputmode="numeric" pattern="[0-9]{6}" minlength="6" maxlength="6" autocomplete="one-time-code" required></div></form>';
      return modal("启用邮箱登录验证","验证邮箱后，后续密码登录将发送一次性验证码。",body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="email-verification-form">启用邮箱验证</button>');
    }
    if(m.kind==="mfa-setup"){
      const body='<p>在身份验证器中添加账户，输入下面的密钥或导入配置链接，再填写当前 6 位验证码。</p>'+
        '<div class="field"><label>身份验证器密钥</label><input readonly value="'+esc(m.secret)+'" autocomplete="off"></div>'+
        '<div class="field"><label>身份验证器配置链接</label><textarea readonly rows="3">'+esc(m.uri)+'</textarea></div>'+
        '<form id="mfa-activate-form"><div class="field"><label for="mfa-activate-code">6 位动态验证码</label>'+
        '<input id="mfa-activate-code" name="code" inputmode="numeric" pattern="[0-9]{6}" minlength="6" maxlength="6" autocomplete="one-time-code" required></div></form>';
      return modal("启用双重认证","身份验证器密钥只在绑定时显示，验证成功后生成恢复码。",body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="mfa-activate-form">验证并启用</button>');
    }
    if(m.kind==="mfa-codes"){
      const list=(m.codes||[]).map(code=>'<li><code>'+esc(code)+'</code></li>').join("");
      const body='<p>请保存下面 6 个一次性恢复码。如果身份验证器无法使用，可以在登录时输入其中一个恢复码。</p>'+
        '<ol style="margin-top:12px">'+list+'</ol><p>恢复码关闭后不会再次显示。</p>';
      return modal("保存双重认证恢复码","建议将恢复码保存在安全的位置。",body,
        '<button type="button" class="btn btn-primary" data-action="close-modal">我已保存恢复码</button>');
    }
    if(m.kind==="mfa-action"){
      const actionName=m.action==="disable"?"解除双重认证":"重新生成恢复码";
      const body='<form id="mfa-confirm-form"><div class="field"><label for="mfa-confirm-code">动态验证码或恢复码</label>'+
        '<input id="mfa-confirm-code" name="code" autocomplete="one-time-code" minlength="6" maxlength="20" required></div></form>';
      return modal(actionName,"确认后将应用当前账户的双重认证设置。",body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="mfa-confirm-form">确认</button>');
    }
    if(m.kind==="audit-note"){
      const body='<form id="audit-note-form">'+
        '<div class="field"><label for="audit-note-input">连接备注</label>'+
        '<textarea id="audit-note-input" name="note" rows="7" maxlength="2000" placeholder="填写连接用途、问题描述或处理记录">'+esc(m.data.note||"")+'</textarea>'+
        '<small>最多 2000 字。内容将显示在连接记录中，并可随 CSV 导出。</small></div></form>';
      return modal("编辑连接备注","连接编号："+m.data.id,body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="button" class="btn btn-primary" data-action="audit-note-save">保存备注</button>');
    }
    if(m.kind==="created-api-token"){
      const body='<div class="field"><label for="api-token-secret">请保存以下令牌</label>'+
        '<input id="api-token-secret" readonly value="'+esc(m.token)+'" autocomplete="off">'+
        '<small>令牌原文仅显示一次，关闭页面后无法再次查看。</small></div>';
      return modal("API 管理令牌已创建","可用于已授权的 RustDesk 客户端命令及自动化管理接口。",body,
        '<button class="btn btn-primary" data-action="copy-api-token">复制令牌</button><button class="btn" data-action="close-modal">关闭</button>');
    }
    if(m.kind==="audit-export"){
      const field=(id,label,type="text")=>'<div class="field"><label for="audit-filter-'+id+'">'+label+'</label>'+
        (["peer_id","from_peer"].includes(id)?pickerField(id,"",false,{},"audit-filter-"+id):
          '<input id="audit-filter-'+id+'" type="'+type+'">')+'</div>';
      const body='<div class="form-grid">'+
        field("from_date","开始日期","date")+field("to_date","结束日期","date")+
        (["audit_conn","audit_file","audit_alarm"].includes(m.kindName)?
          field("peer_id","目标设备 ID"):"")+
        (["audit_conn","audit_file"].includes(m.kindName)?
          field("from_peer","来源设备 ID"):"")+
        (m.kindName==="audit_conn"?'<div class="field"><label for="audit-filter-action">连接事件</label>'+
           '<select id="audit-filter-action"><option value="">全部</option><option value="new">连接建立</option>'+
           '<option value="close">连接结束</option></select></div>':"")+
        '</div>';
      return modal("筛选并导出审计记录","最多导出最近的 10000 条匹配记录。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" data-action="audit-export-submit">下载 CSV</button>');
    }
    if(m.kind==="rules-batch"){
      const revoke=m.action==="revoke";
      const body='<div class="form-grid">'+
        '<div class="field full"><label>选择需要共享的地址簿</label>'+
        pickerField("collection_id","",true,{},"rules-batch-collection")+'</div>'+
        '<div class="field full"><label>搜索并选择目标用户（最多 100 位）</label>'+
        multiPicker("rules-batch-users")+'</div>'+
        (revoke?"":'<div class="field full"><label for="rules-batch-level">共享权限</label>'+
        '<select id="rules-batch-level"><option value="1">只读</option>'+
        '<option value="2">可读写</option><option value="3">完全控制</option></select></div>')+
        '</div>';
      return modal(revoke?"批量撤销地址簿共享":"批量共享地址簿",
        "只能修改您本人拥有的地址簿，服务器会在保存前检查所有目标用户。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" data-action="rules-batch-submit">确认保存</button>');
    }
    if(m.kind==="deployment-batch"){
      const revoke=m.action==="revoked";
      const body='<div class="field"><label>选择需要修改的设备部署记录（最多 200 条）</label>'+
        multiPicker("deployment-batch-ids","deployment_record")+
        '<small>搜索设备名称或设备 ID，可在弹窗中同时选择多条记录。</small></div>';
      return modal(revoke?"批量撤销设备部署":"批量批准设备部署",
        revoke?"撤销后设备将无法在严格登记模式下重新登记。":"选中设备将被批准登记。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" data-action="deployment-batch-submit">确认修改</button>');
    }
    if(m.kind==="contact-batch"){
      const remove=m.action==="remove";
      const body='<div class="field"><label>搜索并选择联系人</label>'+
        multiPicker("contact-batch-ids")+
        '<small>每次最多处理 100 位用户，编号可通过用户列表或现有联系人查看。所有共享关系均属于当前账户。</small></div>';
      return modal(remove?"批量撤销联系人共享":"批量添加联系人",
        remove?"选中联系人将无法再通过联系人关系查看您的设备。":"添加后对方可查看您公开的设备。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" data-action="contact-batch-submit">'+(remove?"确认撤销":"批量添加")+'</button>');
    }
    if(m.kind==="trash-batch"){
      const body='<div class="field"><label>选择回收站记录（最多 200 条）</label>'+
        multiPicker("trash-batch-ids","trash_record")+
        '<small>按设备 ID、备注或主机名搜索需要恢复或永久删除的记录。</small></div>';
      const purging=m.action==="purge";
      return modal(purging?"批量永久删除地址簿条目":"批量恢复地址簿条目",
        purging?"删除后无法恢复，请确认编号。":"整批检查完成后才会恢复记录。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" data-action="trash-batch-submit">'+(purging?"确认永久删除":"批量恢复")+'</button>');
    }
    if(m.kind==="transfer-device"){
      const body='<form id="device-transfer-form"><div class="form-grid">'+
        '<div class="field full"><label>设备 ID</label><div class="device-status-inline">'+deviceStatusDot(m.device.id)+'<input readonly value="'+esc(m.device.id)+'"></div></div>'+
        '<div class="field full"><label>新所有者的用户编号 <span class="required">*</span></label>'+
        pickerField("user_id",m.device.user_id||"",true,m.device,"transfer-user-id")+
        '<small>仅允许将设备转移给当前账户有权管理的用户。</small></div></div></form>';
      return modal("转移设备所有者","设备登记记录将同步更新。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" type="submit" form="device-transfer-form">确认转移</button>');
    }
    if(m.kind==="reset-password"){
      const body='<form id="reset-password-form"><input type="hidden" name="id" value="'+esc(m.user.id)+'"><div class="form-grid"><div class="field full"><label for="new-user-password">新密码 <span class="required">*</span></label><input id="new-user-password" name="password" type="password" required minlength="8" maxlength="72" autocomplete="new-password"></div></div></form>';
      return modal("重设用户密码","账户："+m.user.username,body,'<button class="btn" type="button" data-action="close-modal">取消</button><button class="btn btn-primary" type="submit" form="reset-password-form">保存新密码</button>');
    }
    if(m.kind==="members"||m.kind==="anchors"){
      const anchor=m.kind==="anchors";
      const rows=(m.users||[]).map(u=>'<div class="status-row"><span><strong>'+entityLink("user",u.id,u.nickname||u.username)+'</strong><span class="cell-sub">@'+entityLink("user",u.id,u.username)+' · ID '+esc(u.id)+'</span></span><input style="width:18px;height:18px;accent-color:var(--main)" type="checkbox" name="selected" aria-label="选择用户 '+esc(u.username)+'" value="'+esc(u.id)+'" '+(m.selected.has(u.id)?"checked":"")+'></div>').join("");
      const personalHint="";
      const details='<p class="subtitle">'+(anchor?"只有选中的公共成员及其设备会显示给该组的成员，包括管理员。":"当前成员已经勾选。取消勾选可以移除成员，添加新用户后保存即可一次修改多人。")+'</p>'+personalHint+'<div class="field" style="margin-top:20px"><label for="user-filter">筛选已显示的用户</label><input id="user-filter" placeholder="输入用户名或编号"></div><form id="member-form"><div class="field" style="margin-top:16px"><label>搜索并添加用户</label>'+multiPicker("manual_ids")+'</div><div id="member-choices" style="margin-top:12px">'+rows+'</div></form>';
      const footer='<button class="btn" data-action="close-modal">取消</button><button class="btn btn-primary" data-action="member-replace">'+(anchor?"保存公共成员":"保存群组成员")+'</button>';
      return modal(anchor?"设置公共成员":"批量管理成员",m.group.name,details,footer);
    }
    return "";
  }
  async function openEditor(rowId,action){
    const p=pages[activeKey()];let row={};
    if(action==="edit"){
      row=state.rows.find(r=>rowIdentifier(r)===rowId)||{};
      if(!Object.keys(row).length){toast("未找到记录",true);return;}
    }
    state.modal={kind:"edit",action,data:row};render();
  }
  async function saveEditor(form){
    const p=pages[activeKey()],data={};
    const oldData=state.modal.data;
    for(const f of p.fields.filter(f=>{
      if(state.page==="users"&&state.modal.action==="edit"&&f[0]==="password")return false;
      if(!state.isAdmin&&state.page==="addressbooks"&&f[0]==="user_id")return false;
      return true;
    })){
      const [name,,type="text",required=false]=f;
      const el=form.elements.namedItem(name);
      if(!el)continue;
      if(type==="admin_permissions"){
        const checked=[...form.querySelectorAll('input[name="permissions"]:checked')].map(input=>input.value);
        if(checked.length===0){toast("请至少选择一项管理权限",true);return;}
        data[name]=checked.join(",");
      }
      else if(type==="checkbox")data[name]=el.checked;
      else if(type==="color"){
        const alpha=Number(form.elements.namedItem(name+"_alpha").value);
        if(!Number.isInteger(alpha)||alpha<0||alpha>255){toast("请选择有效的透明度",true);return;}
        data[name]=((alpha*16777216)+parseInt(el.value.slice(1),16))>>>0;
      }
      else if(name==="auto_group_ids")data[name]=el.value.trim();
      else if(lookupKind(name,oldData)){
        const raw=el.value.trim(),kind=lookupKind(name,oldData);
        if(required&&kind!=="all"&&!raw){toast("请先搜索并选择："+f[1],true);return;}
        data[name]=kind==="device"||name==="target_id"?raw:Number(raw||0);
      }
      else if(type==="number"||["status","group_type","rule_type","rule","permission_state"].includes(type)){
        if(el.value.trim()!==""||required)data[name]=Number(el.value||0);
      }else data[name]=el.value.trim();
    }
    if(state.modal.action==="edit"){
      if(oldData.id!==undefined)data.id=oldData.id;
      if(oldData.row_id!==undefined)data.row_id=oldData.row_id;
    }
    try{
      const operation=state.modal.action==="edit"?"update":"create";
      const result=await request(endpoint(activeKey())+"/"+operation,"POST",data);
      if(p.endpoint==="api_token"&&state.modal.action==="create"){
        if(!result?.token)throw Error("服务器未返回新令牌");
        state.modal={kind:"created-api-token",token:result.token};
        render();
        return;
      }
      state.modal=null;
      if(["groups","mygroups","devicegroups","users","policies","collections","adminRoles","controlRoles"].includes(state.page))state.referenceLoaded={};
      await load();
      toast(p.endpoint==="invite"?"邀请邮件已发送":"保存成功");
    }catch(e){toast(e.message,true);}
  }
  async function deleteEntry(id){
    const p=pages[activeKey()],r=state.rows.find(x=>rowIdentifier(x)===id);
    if(!r||!window.confirm("确定删除这条记录吗？"))return;
    try{await request(endpoint(activeKey())+"/delete","POST",{id:r.id,row_id:r.row_id});toast("已删除");await load();}
    catch(e){toast(e.message,true);}
  }
  async function groupMembers(id,anchor){
    const group=state.rows.find(r=>String(r.id)===String(id));
    if(!group)return;
    const personal=state.page==="mygroups";
    try{
      const rootPath=personal?"/my/group":"/group";
      const selected=await request(rootPath+"/"+(anchor?"anchors":"members")+"?group_id="+encodeURIComponent(id));
      const rows=personal?(selected.users||[]):((await request("/user/list?page=1&page_size=500")).list||[]);
      const displayed=new Set(rows.map(x=>Number(x.id)));
      for(const member of selected.users||[]){
        if(!displayed.has(Number(member.id))){rows.push(member);displayed.add(Number(member.id));}
      }
      state.modal={kind:anchor?"anchors":"members",group,personal,selected:new Set((selected.user_ids||[]).map(Number)),users:rows};
      render();
    }catch(e){toast(e.message,true);}
  }
  function parseIds(value){
    if(!value.trim())return [];
    const list=value.split(",").map(x=>x.trim()).filter(Boolean);
    if(list.some(x=>!/^[1-9][0-9]*$/.test(x)))throw Error("用户 ID 需要用逗号分隔的正整数");
    return list.map(Number);
  }
  async function submitMembers(action){
    const m=state.modal,form=document.getElementById("member-form");if(!m||!form)return;
    let ids;try{
      ids=[...new Set([...Array.from(form.querySelectorAll('input[name="selected"]:checked')).map(x=>Number(x.value)),...parseIds(form.elements.manual_ids.value)])];
    }catch(e){toast(e.message,true);return;}
    if(action==="remove"&&!window.confirm("确定从群组移除这 "+ids.length+" 名选中成员吗？"))return;
    if(action==="replace"&&!window.confirm("确定将这个群组的成员设为选中的 "+ids.length+" 人吗？"))return;
    try{
      const base=m.personal?"/my/group":"/group";
      const url=m.kind==="anchors"?base+"/anchors":base+"/members/batch";
      const payload=m.kind==="anchors"?{group_id:m.group.id,user_ids:ids}:{group_id:m.group.id,user_ids:ids,action};
      await request(url,"POST",payload);
      state.modal=null;toast("群组成员已更新");await load();
    }catch(e){toast(e.message,true);}
  }
  async function transferAddressBookCSV(){
    const operation=state.modal?.operation;
    if(state.modal?.kind!=="csv"||!["import","export"].includes(operation))return;
    const ownerField=document.getElementById("csv-owner");
    const owner=ownerField?Number(ownerField.value):null;
    if(ownerField&&(!Number.isSafeInteger(owner)||owner<=0)){toast("请填写有效的所属账户 ID",true);return;}
    const path=endpoint("addressbooks")+"/"+operation+(ownerField?"?user_id="+encodeURIComponent(owner):"");
    const headers={"api-token":state.token,"Accept":operation==="import"?"application/json":"text/csv"};
    let body;
    if(operation==="import"){
      const file=document.getElementById("csv-file")?.files?.[0];
      if(!file){toast("请选择 CSV 文件",true);return;}
      if(file.size>4*1024*1024){toast("CSV 文件不能超过 4 MB",true);return;}
      body=await file.text();
      headers["Content-Type"]="text/csv; charset=utf-8";
    }
    try{
      const reply=await fetch(API+path,{
        method:operation==="import"?"POST":"GET",headers,body,
        cache:"no-store",credentials:"same-origin"
      });
      if(operation==="import"){
        const res=await reply.json();
        if(!reply.ok||res.code!==0)throw Error(res.error||res.message||"地址簿导入失败");
        const result=res.data||{};
        state.modal=null;
        toast("CSV 导入完成：新增 "+(result.created||0)+" 条，更新 "+(result.updated||0)+" 条");
        await load();
      }else{
        if(!reply.ok)throw Error("地址簿导出失败，HTTP "+reply.status);
        const blob=await reply.blob(),url=URL.createObjectURL(blob);
        try{
          const link=document.createElement("a");
          link.href=url;link.download="rustdesk-addressbook.csv";
          document.body.append(link);link.click();link.remove();
        }finally{URL.revokeObjectURL(url);}
        state.modal=null;render();toast("CSV 文件已生成");
      }
    }catch(err){toast(err.message,true);}
  }

  function isValidIP(value){
    const ip=String(value||"").trim();
    if(/^(?:\d{1,3}\.){3}\d{1,3}$/.test(ip))return ip.split(".").every(part=>Number(part)<=255);
    return /^[0-9a-fA-F:]+$/.test(ip)&&ip.includes(":")&&ip.length<=45&&!ip.includes(":::")&&ip.split(":").filter(Boolean).every(part=>part.length<=4);
  }
  async function sendCommand(form){
    const cmd=serverCommandCatalog.find(item=>item.id===state.commandSelected);
    if(!cmd)return;
    let option="",value=String(form.elements.option?.value||form.elements.device_id?.value||"").trim();
    if(cmd.mode.startsWith("fixed-"))option=cmd.mode.slice(6);
    else if(cmd.mode==="ip-pair"){
      const a=form.elements.ip1.value.trim(),b=form.elements.ip2.value.trim();
      if(!isValidIP(a)||!isValidIP(b)){toast("请填写两个有效的 IP 地址",true);return;}
      option=a+" "+b;
    }else if(cmd.mode==="ip"||cmd.mode==="ip-remove"){
      if(!isValidIP(value)){toast("请输入有效的 IPv4 或 IPv6 地址",true);return;}
      option=value+(cmd.mode==="ip-remove"?" -":"");
    }else if(cmd.mode==="identifier"||cmd.mode==="identifier-remove"){
      if(!/^[a-zA-Z0-9_.:-]{1,128}$/.test(value)){toast("请先选择有效的设备 ID",true);return;}
      option=value+(cmd.mode==="identifier-remove"?" -":"");
    }else if(cmd.mode==="number"){
      const num=Number(value);
      if(!value||!Number.isFinite(num)||num<=0||num>1e9||cmd.cmd==="downgrade-threshold"&&num>1||cmd.cmd==="downgrade-start-check"&&!Number.isInteger(num)){toast("请填写符合当前设置范围的正数",true);return;}
      option=String(num);
    }else if(cmd.mode==="servers"){
      const entries=value.split(",").map(x=>x.trim()).filter(Boolean);
      if(!entries.length||entries.length>32||entries.some(x=>!/^[a-zA-Z0-9._:\[\]-]+$/.test(x))){
        toast("请使用英文逗号分隔有效的服务器地址，最多 32 个",true);return;
      }
      option=entries.join(",");
    }
    if(cmd.mode!=="none"&&!option){toast("请填写该操作要求的参数",true);return;}
    const isRead=cmd.mode==="none"&&cmd.cmd!=="reload-geo"||cmd.mode==="identifier"||
      cmd.mode==="ip"&&["blacklist","blocklist","ip-blocker"].includes(cmd.cmd);
    if(!isRead&&!window.confirm("确认执行“"+cmd.title+"”吗？"))return;
    try{
      const response=await request("/rustdesk/execute","POST",{target:cmd.target,cmd:cmd.cmd,option});
      if(!response||typeof response!=="object"||!response.kind)throw Error("服务器返回的数据格式无效");
      let displayed=response;
      const refreshMap={
        "blacklist-add":"blacklist","blacklist-remove":"blacklist",
        "blocklist-add":"blocklist","blocklist-remove":"blocklist"
      };
      const refresh=refreshMap[cmd.cmd]||(
        !isRead&&["relay-servers","always-use-relay","must-login","downgrade-threshold",
          "downgrade-start-check","limit-speed","total-bandwidth","single-bandwidth",
          "ip-blocker","ip-changes"].includes(cmd.cmd)?cmd.cmd:"");
      if(refresh){
        try{
          displayed=await request("/rustdesk/execute","POST",{target:cmd.target,cmd:refresh,option:""});
        }catch{displayed=response;}
      }
      state.serverResults[cmd.id]=displayed;
      state.serverError="";
      render();
      toast(isRead?"查询完成":"操作请求已发送至服务器");
    }catch(err){
      state.serverError=err.message;
      render();
      toast(err.message,true);
    }
  }

  function addMultiPickerValue(picker,id){
    const target=document.getElementById(picker?.dataset.target||"");
    if(!target)return;
    const values=target.value.split(/[,，\s]+/).filter(Boolean);
    if(!values.includes(id))values.push(id);
    target.value=values.join(",");
    const kind=picker.dataset.picker,label=lookupLabels[kind]||"用户";
    const selected=picker.querySelector(".picker-selected");
    selected.innerHTML=values.map(value=>'<button type="button" class="picker-chip" data-action="picker-remove" data-id="'+esc(value)+'" aria-label="移除'+label+' '+esc(value)+'">'+(kind==="device"?deviceStatusDot(value):"")+( ["user_group","deployment_record","trash_record"].includes(kind)?esc(referenceText(kind,value)):esc(label+" "+value))+' ×</button>').join("");
    picker.querySelector(".picker-results").hidden=true;
    picker.querySelector(".picker-selection").textContent="已选择 "+values.length+" 个"+label;
  }
  function openWebClientDevice(id,name){
    const deviceId=String(id||"").trim();
    if(!deviceId)return;
    const params=new URLSearchParams({id:deviceId});
    if(name)params.set("name",String(name).slice(0,160));
    const tab=window.open("/webclient2/?"+params.toString(),"_blank");
    if(tab)tab.opener=null;
    else toast("浏览器阻止了打开新标签页，请在浏览器中允许该操作。",true);
  }
  document.addEventListener("click",async event=>{
    const el=event.target.closest("[data-action],[data-view]");
    if(!el)return;
    if(el.dataset.view){view(el.dataset.view);return;}
    const a=el.dataset.action;
    if(a==="device-connect"){
      openWebClientDevice(el.dataset.deviceId,el.dataset.deviceName);
      return;
    }
    if(a==="entity-detail"){
      await openEntityDetail(el.dataset.entityType,el.dataset.entityId,el.dataset.entityLabel);
      return;
    }
    if(a==="entity-back"){
      state.modal=state.modal?.previous||null;render();return;
    }
    if(a==="entity-admin-edit"){
      const m=state.modal,record=m?.kind==="entity"&&m.entityType==="user"?m.info:null;
      if(!record?.id||!canPageAction("users","edit"))return;
      state.page="users";state.query=record.username||"";state.pageNo=1;
      location.hash="#/users?search="+encodeURIComponent(state.query);
      state.modal=null;render();await load();
      const row=state.rows.find(item=>String(item.id)===String(record.id));
      if(row)await openEditor(String(row.id),"edit");
      return;
    }
    if(a==="entity-open-list"){
      const m=state.modal;if(m?.kind!=="entity"||!m.destination)return;
      view(m.destination,m.destination==="profile"?"":m.entityType==="user"?String(m.info.username||m.entityId):m.entityId);
      return;
    }
    if(a==="refresh"){await load();return;}
    if(a==="server-tab"){
      const tab=el.dataset.tab;
      if(!["overview","id","relay"].includes(tab))return;
      state.serverTab=tab;
      state.commandSelected=tab==="relay"?"relay-usage":"id-relay-read";
      state.serverError="";render();
      return;
    }
    if(a==="server-select"){
      const command=serverCommandCatalog.find(item=>item.id===el.dataset.id);
      if(!command)return;
      state.commandSelected=command.id;state.serverError="";
      render();document.getElementById("server-command-panel")?.scrollIntoView({block:"nearest"});
      return;
    }
    if(a==="picker-open"){await openPickerDialog(el.closest(".entity-picker"));return;}
    if(a==="picker-close"){closePickerDialog();return;}
    if(a==="picker-search"){
      const picker=el.closest(".picker-dialog")||el.closest(".entity-picker");
      if(!picker)return;
      clearTimeout(picker.searchTimer);
      await lookupPicker(picker);
      return;
    }
    if(a==="picker-clear"){
      const picker=el.closest(".entity-picker");
      if(!picker)return;
      clearTimeout(picker.searchTimer);
      picker.dataset.sequence=String(Number(picker.dataset.sequence||0)+1);
      if(picker.dataset.target){
        const target=document.getElementById(picker.dataset.target);
        if(target)target.value="";
        picker.querySelector(".picker-selected").replaceChildren();
      }else picker.querySelector(".picker-id").value="";
      picker.querySelector(".picker-trigger-label").textContent="点击选择"+(lookupLabels[picker.dataset.picker]||"对象");
      picker.querySelector(".picker-results").hidden=true;
      picker.querySelector(".picker-selection").textContent="尚未选择";
      return;
    }
    if(a==="picker-manual-single"){
      const picker=el.closest(".entity-picker"),input=picker?.querySelector(".picker-manual-value");
      if(!picker||!input)return;
      const value=input.value.trim();
      if(!/^[A-Za-z0-9_.:*?@-]{1,253}$/.test(value)){toast("请输入有效的设备 ID",true);return;}
      picker.querySelector(".picker-id").value=value;
      picker.querySelector(".picker-trigger-label").textContent=value;
      picker.querySelector(".picker-results").hidden=true;
      picker.querySelector(".picker-selection").textContent="已填写设备 ID："+value;
      input.value="";return;
    }
    if(a==="picker-manual-add"){
      const picker=el.closest(".entity-picker"),input=picker?.querySelector(".picker-manual-value");
      if(!picker?.dataset.target||!input)return;
      const value=input.value.trim();
      if(!/^[A-Za-z0-9_.:*?@-]{1,253}$/.test(value)){toast("请输入有效的设备 ID 或匹配规则",true);return;}
      addMultiPickerValue(picker,value);input.value="";return;
    }
    if(a==="picker-select"){
      const fromDialog=!!el.closest(".picker-dialog");
      const picker=fromDialog?state.pickerDialog?.picker:el.closest(".entity-picker");
      if(!picker)return;
      const id=el.dataset.id,title=el.dataset.title;
      if(picker.dataset.target){
        addMultiPickerValue(picker,id);
        if(fromDialog){
          const q=state.pickerDialog?.dialog?.querySelector(".picker-query");
          if(q){q.value="";q.focus();}
        }
      }else{
        picker.querySelector(".picker-id").value=id;
        picker.dataset.username=el.dataset.username||"";
        const label=(title||referenceText(picker.dataset.picker,id));
        picker.querySelector(".picker-trigger-label").textContent=label+"（编号 "+id+"）";
        picker.querySelector(".picker-selection").textContent="已选择："+label+"（"+id+"）";
      }
      picker.querySelector(".picker-results").hidden=true;
      if(fromDialog&&!picker.dataset.target)closePickerDialog();
      return;
    }
    if(a==="picker-remove"){
      const picker=el.closest(".entity-picker"),target=document.getElementById(picker?.dataset.target||"");
      if(!picker||!target)return;
      const values=target.value.split(",").filter(v=>v&&v!==el.dataset.id);
      target.value=values.join(",");
      el.remove();
      picker.querySelector(".picker-selection").textContent="已选择 "+values.length+" 个"+(lookupLabels[picker.dataset.picker]||"用户");
      return;
    }
    if(a==="toggle-section"){
      const section=el.dataset.section;
      state.navCollapsed[section]=!state.navCollapsed[section];
      localStorage.setItem("rustdesk.console.nav-collapsed",JSON.stringify(state.navCollapsed));
      render();return;
    }
    if(a==="mobile-sort-dir"){
      const order=state.sort[state.page]||{};
      state.sort[state.page]={key:order.key||pages[activeKey()]?.columns?.[0]?.[0],dir:order.dir===-1?1:-1};
      state.pageNo=1;await load();return;
    }
    if(a==="table-details"){
      const row=document.querySelector('.table-detail-row[data-details-row="'+el.dataset.row+'"]');
      if(row){row.hidden=!row.hidden;el.setAttribute("aria-expanded",String(!row.hidden));el.textContent=row.hidden?"查看详情":"收起详情";}
      return;
    }
    if(a==="table-columns"){state.showAllColumns[state.page]=!state.showAllColumns[state.page];render();return;}
    if(a==="sort-table"){
      const key=el.dataset.column,order=state.sort[state.page]||{};
      if(!pages[activeKey()]?.columns?.some(([column])=>column===key))return;
      state.sort[state.page]={key,dir:order.key===key&&order.dir===1?-1:1};
      state.pageNo=1;await load();return;
    }
    if(a==="setting-save"||a==="setting-reset"){
      const item=state.settings?.items?.find(x=>x.key===el.dataset.id);
      if(!item)return;
      const reset=a==="setting-reset";
      const field=document.getElementById("setting-value-"+item.key);
      const change={key:item.key,enabled:!reset};
      if(!reset){
        if(!field)return;
        if(!item.sensitive||field.value!=="")change.value=field.value;
        if(item.sensitive&&field.value===""&&!item.override_enabled){
          toast("请先填写需要保存的敏感设置值",true);return;
        }
        const notice=[item.name,"保存后需要重新启动 API。",riskNotice(item.key)].filter(Boolean).join("\n");
        if(!window.confirm(notice+"\n确认保存吗？"))return;
      }
      const current=state.settings;
      const payload={bootstrap_version:current.bootstrap_version,
        database_version:current.database_version,changes:[change]};
      try{
        if(!reset)await request("/settings/validate","POST",payload);
        await request(reset?"/settings/overrides/reset":"/settings/overrides",reset?"POST":"PATCH",payload);
        toast(reset?"已关闭网页覆盖，重启后恢复原设置":"已保存覆盖设置，重启 API 后生效");
        await load();
      }catch(e){toast(e.message,true);}
      return;
    }
    if(a==="settings-outbox-category"){
      if(!["push","notification"].includes(el.dataset.category))return;
      state.settingsFailedCategory=el.dataset.category;
      await load();
      return;
    }
    if(a==="settings-outbox-retry"){
      if(!window.confirm("确定重新处理此通知事件吗？"))return;
      try{await request("/settings/outbox/retry","POST",
        {category:el.dataset.category,id:Number(el.dataset.id)});
        toast("已重新提交事件");await load();
      }catch(e){toast(e.message,true);}
      return;
    }

    if(a==="push-new"){state.pushEditing=null;render();return;}
    if(a==="push-edit"){
      const found=state.pushChannels.find(item=>String(item.id)===String(el.dataset.id));
      if(found){state.pushEditing={...found};render();}
      return;
    }
    if(a==="push-delete"){
      const id=Number(el.dataset.id);
      if(!Number.isSafeInteger(id)||id<=0||!window.confirm("确定删除该推送渠道和发送记录吗？"))return;
      try{
        await request("/push/channels/delete","POST",{id});
        state.pushEditing=null;toast("推送渠道已删除");await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="push-test"){
      const id=Number(el.dataset.id);
      if(!Number.isSafeInteger(id)||id<=0)return;
      try{await request("/push/channels/test","POST",{id});toast("测试通知已加入发送队列");await load();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(a==="log-cleanup-preview"||a==="log-cleanup-run"){
      try{
        const input=readLogCleanupInput();
        if(a==="log-cleanup-preview"){
          const params=new URLSearchParams(input);
          state.cleanupRequest={...input,days:input.days||30};
          state.cleanupPreview=await request("/audit/cleanup/preview?"+params.toString());
          render();
        }else{
          if(!state.cleanupPreview||state.cleanupPreview.total<1||
             input.kind!==state.cleanupRequest.kind||input.mode!==state.cleanupRequest.mode||
             (input.mode==="older"&&input.days!==state.cleanupRequest.days)){
            toast("请先查看当前条件下符合清理要求的记录",true);return;
          }
          const category=cleanupKinds.find(([key])=>key===input.kind)?.[1]||"记录";
          if(!window.confirm("确定永久清理 "+category+" 中的 "+state.cleanupPreview.total+" 项历史数据吗？删除后无法恢复。"))return;
          const result=await request("/audit/cleanup","POST",{...input,confirmation:"DELETE_LOGS"});
          state.cleanupPreview=null;
          toast("已清理 "+(result.deleted||0)+" 项历史数据");
          await load();
        }
      }catch(error){toast(error.message,true);}
      return;
    }
    if(a==="push-retry"){
      const id=Number(el.dataset.id);
      if(!Number.isSafeInteger(id)||id<=0)return;
      try{await request("/push/history/retry","POST",{id});toast("消息已重新提交");await load();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(a==="book-entries"||a==="book-collections"){state.bookTab=a==="book-entries"?"entries":"collections";state.pageNo=1;state.query="";await load();return;}
    if(a==="theme"){state.theme=state.theme==="dark"?"light":"dark";document.documentElement.dataset.theme=state.theme;localStorage.setItem("rustdesk.console.theme",state.theme);render();return;}
    if(a==="mobile-open"||a==="mobile-close"){state.mobile=a==="mobile-open";render();return;}
    if(a==="close-modal"){
      if(state.modal?.kind==="entity"&&state.modal.previous){state.modal=state.modal.previous;render();return;}
      if(["public-reset-password","accept-invitation"].includes(state.modal?.kind))location.hash="#/overview";
      state.modal=null;render();return;
    }
    if(a==="passkey-login"){await passkeyLogin();return;}
    if(a==="passkey-register"){await passkeyRegister();return;}
    if(a==="passkey-revoke"){
      const found=state.passkeys.find(item=>String(item.id)===String(el.dataset.id));
      if(!found||!window.confirm("确定移除 Passkey “"+found.name+"”吗？"))return;
      try{
        await request("/passkey/revoke","POST",{id:Number(found.id)});
        state.passkeys=state.passkeys.filter(item=>String(item.id)!==String(found.id));
        render();toast("Passkey 已移除");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="forgot-password"){state.modal={kind:"forgot-password"};render();return;}
    if(a==="logout"){await logout();return;}
    if(a==="submit-access-rule"){
      const form=document.getElementById("edit-form");
      if(form&&form.reportValidity())await saveEditor(form);
      return;
    }
    if(a==="submit-admin-role"){
      const form=document.getElementById("edit-form");
      if(form&&form.reportValidity())await saveEditor(form);
      return;
    }
    if(a==="submit-control-role"){
      const form=document.getElementById("edit-form");
      if(form&&form.reportValidity())await saveEditor(form);
      return;
    }
    if(a==="submit-invitation"){
      const form=document.getElementById("edit-form");
      if(form&&form.reportValidity())await saveEditor(form);
      return;
    }
    if(a==="submit-api-token"){
      const form=document.getElementById("edit-form");
      if(form&&form.reportValidity())await saveEditor(form);
      return;
    }
    if(a==="rules-batch-share"||a==="rules-batch-revoke"){
      state.modal={kind:"rules-batch",action:a==="rules-batch-share"?"share":"revoke"};
      render();return;
    }
    if(a==="rules-batch-submit"){
      if(state.modal?.kind!=="rules-batch")return;
      const collection=document.getElementById("rules-batch-collection");
      const field=document.getElementById("rules-batch-users");
      if(!collection||!field)return;
      const collectionID=Number(collection.value);
      const ids=[...new Set(field.value.trim().split(/[,，\s]+/).filter(Boolean).map(Number))];
      if(!Number.isSafeInteger(collectionID)||collectionID<=0||
        !ids.length||ids.length>100||ids.some(id=>!Number.isSafeInteger(id)||id<=0)){
        toast("请输入有效的地址簿编号以及最多 100 位用户编号",true);return;
      }
      const action=state.modal.action;
      const level=document.getElementById("rules-batch-level");
      const rule=action==="share"?Number(level?.value):0;
      if(action==="share"&&![1,2,3].includes(rule)){toast("共享权限无效",true);return;}
      if(action==="revoke"&&!window.confirm("确定撤销选中用户的地址簿共享吗？"))return;
      try{
        await request("/my/address_book_collection_rule/batch","POST",
          {collection_id:collectionID,target_user_ids:ids,action,rule});
        state.modal=null;toast(action==="share"?"地址簿共享已更新":"地址簿共享已撤销");
        await load();
      }catch(e){toast(e.message,true);}
      return;
    }
    if(a==="contact-batch-add"||a==="contact-batch-remove"){
      state.modal={kind:"contact-batch",action:a==="contact-batch-add"?"add":"remove"};
      render();return;
    }
    if(a==="contact-batch-submit"){
      if(state.modal?.kind!=="contact-batch")return;
      const field=document.getElementById("contact-batch-ids");
      if(!field)return;
      const ids=[...new Set(field.value.trim().split(/[,，\s]+/).filter(Boolean).map(Number))];
      if(!ids.length||ids.length>100||ids.some(id=>!Number.isSafeInteger(id)||id<=0)){
        toast("请输入 1 至 100 个有效的账户编号",true);return;
      }
      const action=state.modal.action;
      if(action==="remove"&&!window.confirm("确认撤销选中用户的设备共享吗？"))return;
      try{
        await request("/contact/batch","POST",{user_ids:ids,action});
        state.modal=null;toast(action==="add"?"联系人已批量添加":"联系人共享已批量撤销");
        await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="trash-batch-restore"||a==="trash-batch-purge"){
      state.modal={kind:"trash-batch",action:a==="trash-batch-purge"?"purge":"restore"};
      render();
      return;
    }
    if(a==="trash-batch-submit"){
      if(state.modal?.kind!=="trash-batch")return;
      const field=document.getElementById("trash-batch-ids");
      if(!field)return;
      const parts=field.value.trim().split(/[,，\s]+/).filter(Boolean);
      const ids=[...new Set(parts.map(Number))];
      if(!ids.length||ids.length>200||ids.some(x=>!Number.isSafeInteger(x)||x<=0)){
        toast("请输入 1 至 200 个有效的记录编号",true);return;
      }
      const purge=state.modal.action==="purge";
      if(purge&&!window.confirm("确认永久删除选中的 "+ids.length+" 条地址簿记录吗？"))return;
      try{
        await request(endpoint("trash")+"/"+(purge?"batchPurge":"batchRestore"),"POST",{row_ids:ids});
        state.modal=null;toast(purge?"已批量永久删除地址簿条目":"已批量恢复地址簿条目");
        await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="trash-restore"||a==="trash-purge"){
      const row=state.rows.find(item=>String(item.row_id)===String(el.dataset.id));
      if(!row)return;
      const purge=a==="trash-purge";
      if(purge&&!window.confirm("此操作会永久删除设备 "+row.id+" 的地址簿记录，确定继续吗？"))return;
      try{
        await request(endpoint("trash")+"/"+(purge?"purge":"restore"),"POST",{row_id:Number(row.row_id)});
        toast(purge?"地址簿条目已永久删除":"地址簿条目已恢复");
        await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="csv-import"||a==="csv-export"){
      state.modal={kind:"csv",operation:a==="csv-import"?"import":"export"};
      render();
      return;
    }
    if(a==="csv-submit"){await transferAddressBookCSV();return;}
    if(a==="email-verification-setup"){
      try{
        const setup=await request("/mfa/email/setup","POST",{});
        state.modal={kind:"email-verification",secret:setup.secret};
        render();toast("邮箱验证码已发送");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="email-verification-disable"){
      if(!window.confirm("关闭后将使用现有的密码登录方式。确认关闭邮箱验证吗？"))return;
      try{
        await request("/mfa/email/disable","POST",{});
        state.emailVerification=await request("/mfa/email/status");
        render();toast("邮箱登录验证已关闭");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="mfa-setup"){
      try{
        const setup=await request("/mfa/setup","POST",{});
        state.modal={kind:"mfa-setup",...setup};render();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="mfa-disable"||a==="mfa-regenerate"){
      state.modal={kind:"mfa-action",action:a==="mfa-disable"?"disable":"regenerate"};render();
      return;
    }
    if(a==="audit-note"){
      const record=state.rows.find(x=>String(x.id)===String(el.dataset.id));
      if(record){state.modal={kind:"audit-note",data:record};render();}
      return;
    }
    if(a==="audit-note-save"){
      if(state.modal?.kind!=="audit-note")return;
      const note=document.getElementById("audit-note-input")?.value;
      try{
        await request("/audit_conn/note","POST",{id:Number(state.modal.data.id),note});
        state.modal=null;toast("连接备注已保存");await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="invitation-revoke"){
      const row=state.rows.find(x=>String(x.id)===String(el.dataset.id));
      if(!row||!window.confirm("确定撤销发送到 "+row.email+" 的邀请吗？"))return;
      try{await request("/invite/revoke","POST",{id:Number(row.id)});toast("邀请已撤销");await load();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(a==="revoke-api-token"){
      const item=state.rows.find(x=>String(x.id)===String(el.dataset.id));
      if(!item||!window.confirm("撤销后使用此令牌的自动部署将停止。确定撤销吗？"))return;
      try{await request("/api_token/delete","POST",{id:Number(item.id)});toast("管理令牌已撤销");await load();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(a==="copy-api-token"){
      const field=document.getElementById("api-token-secret");
      if(!field)return;
      field.focus();field.select();
      try{
        if(window.navigator?.clipboard?.writeText){
          await window.navigator.clipboard.writeText(field.value);
          toast("已复制令牌");
        }else toast("已选择令牌文本，请手动复制");
      }catch{toast("已选择令牌文本，请手动复制");}
      return;
    }
    if(a==="transfer-device"){
      const device=state.rows.find(row=>String(rowIdentifier(row))===String(el.dataset.id));
      if(device){state.modal={kind:"transfer-device",device};render();}
      return;
    }
    if(a==="client-save"){
      await saveClientProfile(document.getElementById("client-profile-form"));
      return;
    }
    if(a==="client-new"){state.clientEditing=null;render();return;}
    if(a==="client-edit"){
      const profile=state.clientProfiles.find(item=>String(item.id)===String(el.dataset.id));
      if(profile){state.clientEditing={...profile};render();}
      return;
    }
    if(a==="client-delete"){
      const id=Number(el.dataset.id);
      if(!Number.isSafeInteger(id)||id<=0||!window.confirm("确定删除此配置吗？历史任务和安装包仍会保留。"))return;
      try{
        await request("/client_builder/profiles/delete","POST",{id});
        if(state.clientEditing?.id===id)state.clientEditing=null;
        toast("客户端配置已删除");await load();
      }catch(e){toast(e.message,true);}
      return;
    }
    if(a==="client-build-local"){
      try{
        const form=document.querySelector("#client-profile-form");
        const input=document.querySelector("#client-local-file");
        const file=input?.files?.[0];
        if(!form||!file)throw Error("请先选择本地 RustDesk 安装文件");
        const local=window.RustDeskLocalBuilder;
        if(!local||typeof local.build!=="function")throw Error("浏览器本地安装包生成组件未加载");
        const settings={};
        for(const key of ["host","public_key","api_server","relay_server"]){
          settings[key]=String(form.elements.namedItem(key)?.value||"").trim();
        }
        const platform=String(form.elements.namedItem("platform")?.value||"");
        const result=await local.build(file,platform,settings);
        const link=document.createElement("a"),url=URL.createObjectURL(result.blob);
        link.href=url;
        link.download=result.name;
        document.body.append(link);
        link.click();
        link.remove();
        setTimeout(()=>URL.revokeObjectURL(url),30000);
        toast("安装文件已在浏览器中生成并下载");
      }catch(error){toast(error.message,true);}
      return;
    }
    if(a==="client-build"){
      const profile_id=Number(el.dataset.id);
      if(!Number.isSafeInteger(profile_id)||profile_id<=0)return;
      try{await request("/client_builder/tasks","POST",{profile_id});toast("安装包任务已创建");await load();}
      catch(e){toast(e.message,true);}
      return;
    }
    if(a==="client-retry"){
      const id=Number(el.dataset.id);
      if(!Number.isSafeInteger(id)||id<=0)return;
      try{await request("/client_builder/tasks/retry","POST",{id});toast("任务已重新提交");await load();}
      catch(e){toast(e.message,true);}
      return;
    }
    if(a==="client-download"){
      const task=state.clientTasks.find(item=>String(item.id)===String(el.dataset.id));
      if(!task||task.status!=="ready")return;
      try{
        const reply=await fetch(API+"/client_builder/tasks/"+encodeURIComponent(task.id)+"/download",{
          headers:{"api-token":state.token,"Accept":"application/octet-stream"},
          cache:"no-store",credentials:"same-origin"
        });
        if(!reply.ok)throw Error("安装包下载失败，HTTP "+reply.status);
        const body=await reply.blob(),url=URL.createObjectURL(body);
        try{
          const link=document.createElement("a");
          link.href=url;link.download=task.filename;
          document.body.append(link);link.click();link.remove();
        }finally{URL.revokeObjectURL(url);}
        toast("安装程序已下载，请确认服务器地址后安装");
      }catch(e){toast(e.message,true);}
      return;
    }
    if(a==="notification-retry"){
      const id=Number(el.dataset.id);
      if(!Number.isSafeInteger(id)||id<=0)return;
      try{await request("/notification/retry","POST",{id});toast("已提交重新发送请求");await load();}
      catch(e){toast(e.message,true);}
      return;
    }
    if(a==="session-revoke"){
      const session=state.rows.find(item=>String(item.id)===String(el.dataset.id));
      if(!session)return;
      if(!window.confirm(session.current?"结束当前登录后，需要重新登录。确定继续吗？":"确定结束选中的登录会话吗？"))return;
      try{
        await request(state.page==="mysessions"?"/my/user_token/revoke":"/user_token/delete","POST",{id:Number(session.id)});
        if(session.current){await logout();return;}
        toast("登录会话已结束");await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="session-disconnect"){
      const deviceId=el.dataset.device, connId=Number(el.dataset.id);
      if(!deviceId||!connId)return;
      if(!window.confirm("确定请求结束设备 "+deviceId+" 的连接 "+connId+" 吗？"))return;
      try{
        await request(state.page==="myactiveSessions"?"/my/session/disconnect":"/session/disconnect","POST",{device_id:deviceId,conn_id:connId});
        toast("结束连接请求已提交，客户端下次发送心跳时会收到通知");
        await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="deployment-batch-approve"||a==="deployment-batch-revoke"){
      state.modal={kind:"deployment-batch",action:a==="deployment-batch-revoke"?"revoked":"approved"};
      render();return;
    }
    if(a==="deployment-batch-submit"){
      if(state.modal?.kind!=="deployment-batch")return;
      const field=document.getElementById("deployment-batch-ids");
      if(!field)return;
      const ids=[...new Set(field.value.trim().split(/[,，\s]+/).filter(Boolean).map(Number))];
      if(!ids.length||ids.length>200||ids.some(id=>!Number.isSafeInteger(id)||id<=0)){
        toast("请输入 1 至 200 个有效的设备部署记录编号",true);return;
      }
      const status=state.modal.action;
      if(status==="revoked"&&!window.confirm("确认批量撤销选中设备的登记许可吗？"))return;
      try{
        await request("/device_deployment/batchStatus","POST",{ids,status});
        state.modal=null;toast("设备部署状态已批量更新");await load();
      }catch(e){toast(e.message,true);}
      return;
    }
    if(a==="deployment-toggle"){
      const item=state.rows.find(x=>String(x.id)===String(el.dataset.id));
      if(!item)return;
      const status=item.status==="approved"?"revoked":"approved";
      const question=status==="revoked"?"撤销后该设备将无法在严格登记模式下重新登记。确认撤销吗？":"确定恢复该设备的登记许可吗？";
      if(!window.confirm(question))return;
      try{await request("/device_deployment/status","POST",{id:Number(item.id),status});toast("设备部署状态已更新");await load();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(a==="webclient"){window.location.assign("/webclient2/");return;}
    if(a==="devices-export"){
      const params=new URLSearchParams();
      if(state.query)params.set("search",state.query);
      const route="/peer/export"+(params.toString()?"?"+params:"");
      try{
        const reply=await fetch(API+route,{
          headers:{"api-token":state.token,"Accept":"text/csv"},
          cache:"no-store",credentials:"same-origin"
        });
        if(!reply.ok)throw Error("设备导出失败，HTTP "+reply.status);
        const data=await reply.blob(),url=URL.createObjectURL(data);
        try{
          const link=document.createElement("a");
          link.href=url;link.download="rustdesk-devices.csv";
          document.body.append(link);link.click();link.remove();
        }finally{URL.revokeObjectURL(url);}
        toast("设备清单 CSV 已生成");
      }catch(error){toast(error.message,true);}
      return;
    }
    if(a==="audit-export"){
      state.modal={kind:"audit-export",kindName:el.dataset.id};render();return;
    }
    if(a==="audit-export-submit"){
      if(state.modal?.kind!=="audit-export")return;
      const filters={};
      for(const name of ["from_date","to_date","peer_id","from_peer","action"]){
        const value=document.getElementById("audit-filter-"+name)?.value?.trim();
        if(value)filters[name]=value;
      }
      const kind=state.modal.kindName;
      state.modal=null;render();await exportAudit(kind,filters);
      return;
    }
    if(a==="captcha"){await captcha();render();return;}
    if(a==="oidc"){await oidc(el.dataset.id);return;}
    if(a==="create"||a==="edit"){await openEditor(el.dataset.id,a);return;}
    if(a==="delete"){await deleteEntry(el.dataset.id);return;}
    if(a==="reset-user-mfa"){
      const member=state.rows.find(r=>String(r.id)===String(el.dataset.id))||(state.modal?.kind==="entity"&&state.modal.entityType==="user"&&String(state.modal.info?.id)===String(el.dataset.id)?state.modal.info:null);
      if(!member||!window.confirm("确认清除用户 "+member.username+" 的 TOTP 和邮箱登录验证吗？此操作将结束该用户现有的登录会话。"))return;
      try{
        await request("/user/mfa/reset","POST",{id:Number(member.id)});
        state.modal=null;toast("用户双重认证已清除，现有登录会话已结束");
        await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="reset-password"){
      const user=state.rows.find(r=>String(r.id)===String(el.dataset.id))||(state.modal?.kind==="entity"&&state.modal.entityType==="user"&&String(state.modal.info?.id)===String(el.dataset.id)?state.modal.info:null);
      if(user){state.modal={kind:"reset-password",user};render();}
      return;
    }
    if(a==="members"||a==="anchors"){await groupMembers(el.dataset.id,a==="anchors");return;}
    if(a==="lookup-user"){
      const m=state.modal,field=document.getElementById("lookup-username");
      if(!m?.personal||!field)return;
      const username=field.value.trim();
      if(username.length<2){toast("请输入完整的用户名",true);return;}
      try{
        const selected=document.querySelectorAll('#member-choices input[name="selected"]:checked');
        m.selected=new Set([...selected].map(x=>Number(x.value)));
        const found=await request("/my/group/user/lookup?username="+encodeURIComponent(username));
        if(!m.users.some(x=>Number(x.id)===Number(found.id)))m.users.push(found);
        m.selected.add(Number(found.id));
        toast("已选中成员："+found.username);
        render();
      }catch(e){toast(e.message,true);}
      return;
    }
    if(a.startsWith("member-")){await submitMembers(a.replace("member-",""));return;}
    if(a==="contact-delete"){
      if(!window.confirm("撤销对该联系人的设备共享吗？"))return;
      try{await request("/contact/delete","POST",{user_id:Number(el.dataset.id)});toast("已撤销共享");await load();}catch(e){toast(e.message,true);}return;
    }
    if(a==="contact-devices"){state.page="shared";state.sourceType="contact";state.sourceId=el.dataset.id;location.hash="#/shared";render();await load();return;}
    if(a==="prev"&&state.pageNo>1){state.pageNo--;await load();return;}
    if(a==="next"&&state.pageNo*state.pageSize<state.total){state.pageNo++;await load();return;}
  });
  async function saveClientProfile(f){
    if(!f||!f.reportValidity())return;
    const values=new FormData(f),form={};
    for(const name of ["name","platform","version","host","public_key","api_server","relay_server"]){
      form[name]=String(values.get(name)||"").trim();
    }
    const id=Number(values.get("id"));
    if(Number.isSafeInteger(id)&&id>0)form.id=id;
    try{
      await request("/client_builder/profiles","POST",form);
      state.clientEditing=null;
      toast("客户端配置已保存");
      await load();
    }catch(e){toast(e.message,true);}
  }
  document.addEventListener("submit",async event=>{
    const f=event.target;
    const formID=f.getAttribute("id");
    if(!["login-form","edit-form","contact-form","command-form","password-form","notification-form","client-profile-form","push-channel-form","device-transfer-form","reset-password-form","retention-form","mail-settings-form","mail-test-form","forgot-password-form","public-password-reset-form","accept-invitation-form","mfa-login-form","mfa-activate-form","mfa-confirm-form","email-verification-form"].includes(formID))return;
    event.preventDefault();
    if(formID==="mfa-login-form"){
      const pending=state.modal;
      if(pending?.kind!=="mfa-login")return;
      try{
        const result=await request("/login","POST",{
          username:pending.username,
          type:pending.method==="email_check"?"email_code":"tfa_code",
          secret:pending.secret,tfaCode:f.elements.code.value.trim(),
          verificationCode:f.elements.code.value.trim(),platform:"web"
        });
        if(!result?.token)throw Error("登录未返回访问令牌");
        state.modal=null;state.token=result.token;
        sessionStorage.setItem("rustdesk.console.token",result.token);
        state.user=result;state.isAdmin=Array.isArray(result.route_names)&&result.route_names.includes("*");
        if(returnToWebClient())return;
        state.page="overview";location.hash="#/overview";render();await load();
        toast("登录成功");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="email-verification-form"){
      if(state.modal?.kind!=="email-verification")return;
      try{
        await request("/mfa/email/enable","POST",{
          secret:state.modal.secret,code:f.elements.code.value.trim()
        });
        state.emailVerification=await request("/mfa/email/status");
        state.modal=null;render();toast("邮箱登录验证已启用");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="mfa-activate-form"){
      try{
        const result=await request("/mfa/enable","POST",{code:f.elements.code.value.trim()});
        if(!Array.isArray(result?.recovery_codes))throw Error("恢复码生成失败");
        state.mfa=await request("/mfa/status");
        state.modal={kind:"mfa-codes",codes:result.recovery_codes};
        render();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="mfa-confirm-form"){
      const action=state.modal?.action;
      if(!["disable","regenerate"].includes(action))return;
      try{
        const result=await request(action==="disable"?"/mfa/disable":"/mfa/recovery/regenerate",
          "POST",{code:f.elements.code.value.trim()});
        state.mfa=await request("/mfa/status");
        state.modal=action==="regenerate"?{kind:"mfa-codes",codes:result.recovery_codes}:null;
        render();
        if(action==="disable")toast("双重认证已解除");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="accept-invitation-form"){
      const password=f.elements.password.value;
      if(password!==f.elements.confirm_password.value){
        toast("两次输入的密码不一致",true);return;
      }
      try{
        await request("/invite/accept","POST",{
          token:state.modal?.token,
          username:f.elements.username.value.trim(),
          nickname:f.elements.nickname.value.trim(),
          password
        });
        state.modal=null;location.hash="#/overview";render();
        toast("账户已创建，请使用新账户登录");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="forgot-password-form"){
      try{
        await request("/password/forgot","POST",{email:f.elements.email.value.trim()});
        state.modal=null;render();
        toast("如果邮箱已登记，将收到密码恢复邮件");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="public-password-reset-form"){
      const password=f.elements.password.value;
      if(password!==f.elements.confirm_password.value){
        toast("两次输入的新密码不一致",true);return;
      }
      try{
        await request("/password/reset","POST",{token:state.modal?.token,password});
        state.modal=null;
        location.hash="#/overview";
        render();
        toast("密码已更新，请使用新密码登录");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="mail-settings-form"){
      const values=Object.fromEntries(new FormData(f).entries());
      values.enabled=f.elements.enabled.checked;
      values.clear_password=f.elements.clear_password.checked;
      values.port=Number(values.port);
      try{state.mail=await request("/mail/settings","POST",values);toast("邮件设置已保存");render();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="mail-test-form"){
      try{await request("/mail/test","POST",{to:f.elements.to.value.trim()});toast("测试邮件已发送");}
      catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="retention-form"){
      const values=Object.fromEntries(new FormData(f).entries());
      for(const key of Object.keys(values))values[key]=Number(values[key]);
      try{state.retention=await request("/audit/retention","POST",values);toast("日志保留设置已保存");render();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="login-form")await handleLogin(f);
    if(formID==="client-profile-form"){await saveClientProfile(f);return;}
    if(formID==="notification-form"){
      const events=[...f.querySelectorAll('input[name="event"]:checked')].map(x=>x.value);
      try{
        await request("/notification/settings","POST",{
          enabled:f.elements.enabled.checked,
          recipients:f.elements.recipients.value.trim(),
          events:events.join(",")
        });
        toast("邮件通知设置已保存");await load();
      }catch(e){toast(e.message,true);}
      return;
    }
    if(formID==="edit-form")await saveEditor(f);
    if(formID==="device-transfer-form"){
      if(state.modal?.kind!=="transfer-device")return;
      const rowID=Number(state.modal.device.row_id),userID=Number(f.elements.user_id.value);
      if(!Number.isSafeInteger(rowID)||rowID<=0||!Number.isSafeInteger(userID)||userID<=0)return;
      try{
        await request("/peer/transfer","POST",{row_id:rowID,user_id:userID});
        state.modal=null;toast("设备所有者已更新");await load();
      }catch(e){toast(e.message,true);}
      return;
    }
    if(formID==="reset-password-form"){
      const id=state.modal?.user.id, password=f.elements.password.value;
      if(!id||password.length<8)return;
      try{
        await request("/user/changePwd","POST",{id,password});
        state.modal=null;toast("用户密码已更新");await load();
      }catch(e){toast(e.message,true);}
    }
    if(formID==="contact-form"){
      const picker=f.querySelector(".entity-picker"),username=picker?.dataset.username||"";
      if(!username){toast("请先搜索并选择要添加的用户",true);return;}
      try{await request("/contact/add","POST",{username});toast("已向联系人开放您的设备");await load();}catch(e){toast(e.message,true);}
    }
    if(formID==="command-form")await sendCommand(f);
    if(formID==="push-channel-form"){
      const input=new FormData(f);
      const kind=String(f.elements.kind.value);
      const values={
        id:Number(input.get("id"))||0,
        kind,name:String(input.get("name")||"").trim(),
        enabled:f.elements.enabled.checked,
        events:[...f.querySelectorAll('input[name="event"]:checked')].map(item=>item.value),
        base_url:kind==="messagepusher"?String(input.get("mp_base_url")||""):
          kind==="serverchan"?String(input.get("sc_base_url")||""):"",
        api_key:kind==="messagepusher"?String(input.get("mp_api_key")||""):
          kind==="serverchan"?String(input.get("sc_api_key")||""):"",
        smtp_host:String(input.get("smtp_host")||""),
        smtp_port:Number(input.get("smtp_port")||587),
        smtp_security:String(input.get("smtp_security")||"starttls"),
        smtp_username:String(input.get("smtp_username")||""),
        smtp_password:String(input.get("smtp_password")||""),
        smtp_from:String(input.get("smtp_from")||""),
        smtp_to:String(input.get("smtp_to")||"")
      };
      try{
        await request("/push/channels","POST",values);
        state.pushEditing=null;toast("推送渠道已保存");await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(formID==="password-form"){
      try{
        await request("/user/changeCurPwd","POST",Object.fromEntries(new FormData(f).entries()));
        toast("密码已修改，请重新登录");await logout();
      }catch(e){toast(e.message,true);}
    }
  });
  document.addEventListener("input",event=>{
    const el=event.target;
    if(el.classList.contains("picker-query")&&el.closest(".picker-dialog")){
      const dialog=el.closest(".picker-dialog");
      clearTimeout(dialog.searchTimer);
      dialog.searchTimer=setTimeout(()=>lookupPicker(dialog),250);
    }
    if(el.id==="settings-search"){state.settingsSearch=el.value;
      document.querySelectorAll("[data-setting-item]").forEach(node=>{
        const item=state.settings?.items?.find(x=>x.key===node.dataset.settingItem);
        const text=[item?.key,item?.name,item?.description,item?.environment].join(" ").toLowerCase();
        node.hidden=!text.includes(el.value.toLowerCase());
      });
    }
    if(el.id==="table-search"){
      state.query=el.value;clearTimeout(debounce);
      const p=pages[activeKey()];
      if(state.page==="shared"||p.search){
        debounce=setTimeout(()=>{state.pageNo=1;load();},350);
      }else{
        const pos=el.selectionStart;const value=el.value;render();
        const restored=document.getElementById("table-search");if(restored){restored.focus();restored.value=value;restored.setSelectionRange(pos,pos);}
      }
    }
    if(el.id==="user-filter"){
      const q=el.value.toLowerCase();
      document.querySelectorAll("#member-choices .status-row").forEach(row=>{row.style.display=row.textContent.toLowerCase().includes(q)?"":"none";});
    }
  });
  document.addEventListener("change",async event=>{
    const el=event.target;
    if(["type","source_type","target_type","scope_type"].includes(el.name)&&el.closest("#edit-form")){
      const form=el.closest("#edit-form");
      form.querySelectorAll('.entity-picker[data-picker-dep="'+el.name+'"]').forEach(picker=>{
        const current=Object.fromEntries(new FormData(form));
        const kind=lookupKind(picker.dataset.pickerField,current);
        picker.dataset.picker=kind;
        picker.querySelector(".picker-id").value=kind==="all"?(picker.dataset.pickerField==="target_id"?"":"0"):"";
        picker.querySelector(".picker-trigger-label").textContent=kind==="all"?"适用于全部对象":"点击选择"+(lookupLabels[kind]||"对象");
        picker.querySelectorAll('[data-action="picker-open"],[data-action="picker-clear"]').forEach(btn=>btn.disabled=kind==="all");
        picker.querySelector(".picker-results").hidden=true;
        picker.querySelector(".picker-selection").textContent=kind==="all"?"适用于全部对象":"请搜索并选择"+(lookupLabels[kind]||"对象");
      });
    }
    if(el.id==="table-sort-column"){state.sort[state.page]={key:el.value,dir:1};state.pageNo=1;await load();return;}
    if(el.id==="mobile-sort-column"){
      state.sort[state.page]={key:el.value,dir:1};
      state.pageNo=1;await load();return;
    }
    if(el.id==="settings-category"){state.settingsCategory=el.value;render();return;}
    if(el.id==="push-kind"){
      document.querySelectorAll("[data-push-kind]").forEach(item=>{
        item.hidden=item.dataset.pushKind!==el.value;
      });
    }
    if(el.id==="shared-source-type"){state.sourceType=el.value;state.sourceId="0";state.pageNo=1;await load();}
    if(el.id==="shared-source-id"){if(el.value==="0"){state.sourceId="0";}else{const parts=el.value.split(":");state.sourceType=parts[0];state.sourceId=parts[1];}state.pageNo=1;await load();}
  });
  window.addEventListener("hashchange",()=>{
    const hash=location.hash||"";
    for(const [prefix,kind] of [["#/password-reset?","public-reset-password"],["#/invite?","accept-invitation"]]){
      if(!hash.startsWith(prefix))continue;
      const token=new URLSearchParams(hash.slice(hash.indexOf("?")+1)).get("token");
      if(token){
        state.token="";state.user=null;sessionStorage.removeItem("rustdesk.console.token");
        state.modal={kind,token};render();
      }
      return;
    }
    if(!state.user)return;
    const viewName=hash.replace(/^#\/?/,"").split("?")[0];
    const params=new URLSearchParams(hash.split("?")[1]||"");
    const type=params.get("entity"),id=params.get("value");
    if(available(viewName)&&state.page!==viewName){
      state.page=viewName;
      state.pageNo=1;
      state.query=params.get("search")||"";
      state.sourceType="all";
      state.sourceId="0";
      state.mobile=false;
      state.modal=null;
      state.error="";
      if(viewName==="addressbooks")state.bookTab="entries";
      render();
      load().then(()=>{
        if(["device","user"].includes(type)&&id)openEntityDetail(type,id,id);
      });
      return;
    }
    if(["device","user"].includes(type)&&id&&
      !(state.modal?.kind==="entity"&&state.modal.entityType===type&&state.modal.entityId===id)){
      openEntityDetail(type,id,id);
    }
  });
  bootstrap().catch(e=>{toast(e.message,true);render();});
})();
