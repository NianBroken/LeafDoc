// 服务端步骤是唯一进度来源，页面只显示当前步骤，不持久保存敏感输入。
const names = [
  "安装令牌",
  "数据库地址",
  "连接身份",
  "目标数据库",
  "管理员",
  "完成安装",
];
const hints = [
  "请输入当前实例的安装令牌。可在安装输出或实例的 setup-token 文件中找到。",
  "填写 MySQL 的主机与端口。此步检查网络连通性，下一步验证账号。",
  "使用获授权的 MySQL 账号连接服务。数据库尚未创建也可以继续。",
  "填写供 LeafDoc 使用的数据库名。我们会检查现有结构，或在完成安装时自动创建。",
  "新安装请设置管理员资料。已有实例须使用原管理员账号和密码，邮箱沿用原记录。",
  "核对配置后完成安装。数据库准备、读写检查和系统配置保存均通过后才能进入系统。",
];
const endpoints = ["token", "address", "credentials", "database", "admin"];
const form = document.querySelector("#setup"),
  fields = document.querySelector("#fields"),
  feedback = document.querySelector("#feedback");
let state = { step: 0, revision: 0, config: {} },
  busy = false;
const field = (name, label, type = "text", help = "") => {
  const box = document.createElement("div"),
    l = document.createElement("label"),
    input = document.createElement("input");
  l.htmlFor = name;
  l.textContent = label;
  input.id = name;
  input.name = name;
  input.type = type;
  input.autocomplete = type === "password" ? "off" : "off";
  input.required = !["password", "ca"].includes(name);
  if (state.config?.[name] && type !== "password")
    input.value = state.config[name];
  box.append(l, input);
  if (help) {
    const p = document.createElement("p");
    p.className = "help";
    p.textContent = help;
    box.append(p);
  }
  fields.append(box);
  return input;
};
function message(text, status = "passed") {
  feedback.textContent = text;
  feedback.dataset.state = status;
  document.querySelector("#badge").textContent =
    {
      failed: "校验失败",
      blocked: "暂时无法检查",
      passed: "校验通过",
      checking: "正在校验",
      confirmation: "等待确认",
    }[status] || "等待填写";
}
function render() {
  const step = state.step;
  document.querySelector("#confirmDatabase").required = false;
  document.querySelector("#confirmDatabase").value = "";
  document.querySelector("#confirmation").hidden = true;
  document.querySelector("#actions").hidden = false;
  fields.replaceChildren();
  feedback.textContent = "";
  document.querySelector("#badge").textContent = "等待填写";
  document.querySelector("#title").textContent = names[step];
  document.querySelector("#hint").textContent = hints[step];
  document.querySelector("#progress").textContent =
    `第 ${step + 1} 步，共 6 步`;
  document.querySelector("#bar").style.width = `${((step + 1) / 6) * 100}%`;
  const list = document.querySelector("#steps");
  list.replaceChildren();
  names.forEach((name, index) => {
    const li = document.createElement("li"),
      n = document.createElement("span"),
      label = document.createElement("span");
    li.className = index < step ? "passed" : index === step ? "current" : "";
    if (index === step) li.setAttribute("aria-current", "step");
    n.className = "number";
    n.textContent = index < step ? "✓" : index + 1;
    label.className = "step-name";
    label.textContent = name;
    li.append(n, label);
    list.append(li);
  });
  document.querySelector("#back").hidden = step < 2;
  document.querySelector("#next").textContent =
    step === 5 ? "完成安装" : "验证并继续 →";
  if (step === 0) field("setupToken", "安装令牌", "password");
  if (step === 1) {
    field(
      "host",
      "数据库地址",
      "text",
      "填写主机名或 IP 地址，不包含协议和端口。",
    );
    const port = field("port", "数据库端口", "number");
    port.value = state.config?.port || 3306;
    port.min = 1;
    port.max = 65535;
  }
  if (step === 2) {
    field("user", "数据库账号");
    field(
      "password",
      "数据库密码",
      "password",
      "密码允许为空，请按数据库的实际配置填写。",
    );
    const label = document.createElement("label");
    label.className = "check";
    const check = document.createElement("input");
    check.type = "checkbox";
    check.name = "tls";
    check.id = "tls";
    check.checked = !!state.config?.tls;
    label.append(check, document.createTextNode("使用 TLS 验证数据库证书"));
    fields.append(label);
    const ca = field(
      "ca",
      "自定义 CA 证书",
      "text",
      "留空使用系统 CA。启用 TLS 时验证服务器名称与证书。",
    );
    const area = document.createElement("textarea");
    area.id = ca.id;
    area.name = ca.name;
    area.placeholder = "粘贴 PEM 格式证书";
    ca.replaceWith(area);
  }
  if (step === 3)
    field(
      "database",
      "数据库名",
      "text",
      "不存在时自动创建。已有且兼容的数据库会保留全部数据。",
    );
  if (step === 4) {
    field("admin", "管理员账号");
    const email = field("email", "管理员邮箱", "email");
    email.required = state.inspection?.state !== "compatible";
    const pw = field(
      "adminPassword",
      "管理员密码",
      "password",
      "8 至 72 字节。接入已有实例时填写原管理员密码。",
    );
    pw.minLength = 8;
  }
  if (step === 5) {
    const dl = document.createElement("dl");
    dl.className = "summary";
    for (const [label, value] of [
      ["服务器", `${state.config.host}:${state.config.port}`],
      ["数据库", state.config.database],
      ["数据库账号", state.config.user],
      ["连接加密", state.config.tls ? "TLS 证书验证" : "未启用 TLS"],
      ["管理员", state.config.admin],
      [
        "数据库操作",
        state.format
          ? "清空全部对象并重建"
          : state.inspection?.state === "compatible"
            ? "保留现有数据"
            : "自动创建所需结构",
      ],
    ]) {
      const dt = document.createElement("dt"),
        dd = document.createElement("dd");
      dt.textContent = label;
      dd.textContent = value;
      dl.append(dt, dd);
    }
    fields.append(dl);
  }
}
async function request(path, data) {
  const response = await fetch(base + "/api/setup" + path, {
    method: data ? "POST" : "GET",
    headers: data
      ? { "Content-Type": "application/json", "X-LeafDoc-Setup": "1" }
      : {},
    body: data
      ? JSON.stringify({ ...data, revision: state.revision })
      : undefined,
  });
  const text = await response.text();
  let result;
  try {
    result = JSON.parse(text);
  } catch {
    result = { message: text };
  }
  if (!response.ok) {
    if (result.revision !== undefined) state.revision = result.revision;
    if (response.status === 401) {
      state = { step: 0, revision: 0, config: {} };
      render();
    }
    if (result.step !== undefined && result.step !== state.step) {
      state.step = result.step;
      render();
    }
    throw Object.assign(Error(result.message || "请求失败，请重试"), {
      state: result.state || "failed",
    });
  }
  return result;
}
form.addEventListener("submit", async (event) => {
  event.preventDefault();
  if (busy) return;
  busy = true;
  const buttons = [...form.querySelectorAll("button")];
  buttons.forEach((b) => (b.disabled = true));
  message(
    state.step === 5
      ? "正在准备数据库并保存系统配置，请勿重复提交。"
      : "正在校验当前步骤，请等待。",
    "checking",
  );
  try {
    const values = Object.fromEntries(new FormData(form));
    fields.inert = true;
    if (values.port) values.port = Number(values.port);
    if (state.step === 2) values.tls = form.elements.tls.checked;
    const next = await request(
      state.step === 5 ? "" : "/" + endpoints[state.step],
      values,
    );
    if (next.initialized) {
      message("安装完成，正在进入 LeafDoc。");
      location.assign(base + "/");
      return;
    }
    if (next.state === "confirmation") {
      state.inspection = next.inspection;
      document.querySelector("#confirmation").hidden = false;
      document.querySelector("#actions").hidden = true;
      document.querySelector("#target").textContent =
        `目标：${state.config.host}:${state.config.port} / ${values.database}`;
      document.querySelector("#differences").textContent = [
        ...next.inspection.differences,
        "",
        "对象清单",
        ...next.inspection.objects,
      ].join("\n");
      document.querySelector("#confirmDatabase").required = true;
      message(next.message, "confirmation");
    } else {
      state = next;
      render();
      message("上一步校验通过，请继续填写。");
      document.querySelector("#title").focus();
    }
  } catch (error) {
    message(error.message, error.state);
  } finally {
    busy = false;
    fields.inert = false;
    buttons.forEach((b) => (b.disabled = false));
  }
});
document.querySelector("#back").addEventListener("click", async () => {
  if (busy) return;
  busy = true;
  try {
    state = await request("/back", { step: state.step - 1 });
    render();
    message("修改此步后，后续内容需要重新校验。", "passed");
  } catch (error) {
    message(error.message, error.state);
  } finally {
    busy = false;
  }
});
document.querySelector("#decline").addEventListener("click", () => {
  document.querySelector("#confirmation").hidden = true;
  document.querySelector("#confirmDatabase").required = false;
  document.querySelector("#confirmDatabase").value = "";
  document.querySelector("#actions").hidden = false;
  message(
    "未授权清空，现有数据已保留。请更换数据库或修正结构后重新检查。",
    "failed",
  );
});
// 初始化时只恢复服务端确认的状态，刷新不会重复执行数据库操作。
render();
request("/status")
  .then((value) => {
    state = value;
    render();
  })
  .catch((error) => message(error.message, "blocked"));
