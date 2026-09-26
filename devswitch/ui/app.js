(function () {
  var state = {
    page: "node",
    current: {},
    runtimes: [],
    issues: [],
    selected: "",
    flash: null,
    version: "",
    paths: {}
  };

  function send(msg) {
    var payload = JSON.stringify(msg);
    if (window.webkit && window.webkit.messageHandlers && window.webkit.messageHandlers.devswitch) {
      window.webkit.messageHandlers.devswitch.postMessage(payload);
      return;
    }
    var api = window.pywebview && window.pywebview.api;
    if (api && api.send) {
      api.send(payload);
      return;
    }
    window.addEventListener("pywebviewready", function () {
      if (window.pywebview && window.pywebview.api) window.pywebview.api.send(payload);
    }, { once: true });
  }

  function esc(text) {
    return String(text == null ? "" : text)
      .replace(/&/g, "&amp;")
      .replace(/</g, "&lt;")
      .replace(/>/g, "&gt;")
      .replace(/"/g, "&quot;");
  }

  var LOGO_NODE =
    '<svg viewBox="0 0 32 32"><path fill="#5FA04E" d="M16 2.1 28.7 9.4v13.2L16 29.9 3.3 22.6V9.4L16 2.1z"/><path fill="#3F7E33" d="M16 2.1v27.8l12.7-7.3V9.4L16 2.1z"/><path fill="#8CC84B" d="M16 6.4 24 11v10l-8 4.6L8 21V11l8-4.6z"/><path fill="#fff" d="M14.3 13.2c0-1.3.9-2.1 2.4-2.1 1.1 0 1.9.4 2.4 1.1l-1.2.8c-.3-.4-.6-.6-1.2-.6-.5 0-.8.3-.8.7 0 .4.3.6 1 .8l.8.2c1.4.4 2.1 1.1 2.1 2.3 0 1.4-1.1 2.3-2.7 2.3-1.3 0-2.3-.5-2.8-1.4l1.3-.8c.3.6.8.9 1.5.9.6 0 1-.3 1-.8 0-.4-.3-.6-1-.8l-.8-.2c-1.4-.4-2-1.1-2-2.4z"/></svg>';
  var LOGO_JAVA =
    '<svg viewBox="0 0 32 32"><path fill="none" stroke="#E76F00" stroke-width="1.6" stroke-linecap="round" d="M12 4.5c0 1.8 2 1.8 2 3.6s-2 1.8-2 3.6"/><path fill="none" stroke="#E76F00" stroke-width="1.6" stroke-linecap="round" d="M16 3.5c0 1.8 2 1.8 2 3.6s-2 1.8-2 3.6"/><path fill="#E76F00" d="M8.5 13h13v7.2a5.8 5.8 0 0 1-5.8 5.8h-1.4A5.8 5.8 0 0 1 8.5 20.2V13z"/><path fill="none" stroke="#E76F00" stroke-width="1.8" stroke-linecap="round" d="M21.5 15.2h2.2a2.6 2.6 0 1 1 0 5.2h-2.2"/><rect x="8" y="27" width="14.5" height="1.8" rx=".9" fill="#E76F00"/></svg>';

  function logoHtml(kind) {
    return '<span class="logo">' + (kind === "java" ? LOGO_JAVA : LOGO_NODE) + "</span>";
  }

  function sourceText(source) {
    if (source === "system") return "系统安装";
    if (source === "imported") return "手动导入";
    return "本机目录";
  }

  function currentOf(page) {
    return (state.current || {})[page] || null;
  }

  function isActive(item) {
    var cur = currentOf(item.tool);
    return !!(cur && cur.home === item.home);
  }

  function selectedItem() {
    var list = state.runtimes || [];
    for (var i = 0; i < list.length; i++) {
      if (list[i].home === state.selected) return list[i];
    }
    return list[0] || null;
  }

  function toast(text, kind) {
    var el = document.getElementById("toast");
    el.textContent = text;
    el.className = kind === "error" ? "error" : "";
    el.hidden = false;
    clearTimeout(toast._t);
    toast._t = setTimeout(function () {
      el.hidden = true;
    }, 3200);
  }

  function renderTabs() {
    var tabs = document.querySelectorAll(".tab");
    for (var i = 0; i < tabs.length; i++) {
      tabs[i].classList.toggle("active", tabs[i].getAttribute("data-page") === state.page);
    }
    var cur = currentOf(state.page);
    var tag = document.getElementById("active-tag");
    if (state.page === "doctor") {
      tag.className = "tag " + (state.issues && state.issues.length ? "warn" : "success");
      tag.textContent = state.issues && state.issues.length ? state.issues.length + " 项" : "正常";
    } else if (cur) {
      tag.className = "tag success";
      tag.textContent = cur.version;
    } else {
      tag.className = "tag warn";
      tag.textContent = "未选择";
    }
    document.getElementById("btn-import").hidden = state.page === "doctor";
    var doctorBtn = document.getElementById("btn-doctor");
    if (doctorBtn) doctorBtn.classList.toggle("active", state.page === "doctor");
  }

  function renderSider() {
    var sider = document.getElementById("sider");
    if (state.page === "doctor") {
      if (!state.issues || !state.issues.length) {
        sider.innerHTML = '<div class="empty">没有发现问题</div>';
        return;
      }
      sider.innerHTML = state.issues
        .map(function (issue, index) {
          var selected = String(index) === String(state.selected);
          return (
            '<div class="card' +
            (selected ? " selected" : "") +
            '" data-select="' +
            index +
            '">' +
            '<div class="card-title"><div class="logo doctor">!</div>' +
            '<span class="name">' +
            esc(issue.level === "error" ? "错误" : issue.level === "info" ? "提示" : "注意") +
            "</span></div>" +
            '<span class="summary">' +
            esc(issue.message) +
            "</span></div>"
          );
        })
        .join("");
      return;
    }

    var items = state.runtimes || [];
    if (!items.length) {
      var kind = { node: "Node.js", java: "Java", maven: "Maven", gradle: "Gradle" }[state.page] || state.page;
      sider.innerHTML =
        '<div class="empty">本机没有发现 ' +
        kind +
        '<br/><button class="btn" id="empty-scan">扫描本机</button> ' +
        '<button class="btn" id="empty-import">导入目录</button></div>';
      return;
    }
    sider.innerHTML = items
      .map(function (item) {
        var active = isActive(item);
        var selected = item.home === state.selected;
        return (
          '<div class="card' +
          (selected ? " selected" : "") +
          '" data-select="' +
          esc(item.home) +
          '">' +
          '<div class="card-title">' +
          logoHtml(item.tool) +
          '<span class="name">' +
          esc(item.label || item.version) +
          "</span>" +
          (active ? '<span class="tag success">激活</span>' : "") +
          "</div>" +
          '<span class="summary">' +
          esc(sourceText(item.source) + " · " + item.home) +
          "</span>" +
          '<div class="actions">' +
          (active
            ? ""
            : '<button class="btn-primary" data-use="' + esc(item.home) + '">切换</button>') +
          '<button class="btn-tiny" data-select="' +
          esc(item.home) +
          '">详情</button>' +
          "</div></div>"
        );
      })
      .join("");
  }

  function renderContent() {
    var content = document.getElementById("content");
    if (state.page === "doctor") {
      var issue = (state.issues || [])[parseInt(state.selected, 10) || 0];
      content.innerHTML =
        '<div class="detail">' +
        "<h2>PATH 与冲突</h2>" +
        '<div class="field"><div class="field-label">说明</div><div class="field-value">' +
        esc(
          issue
            ? issue.message
            : "检查 shim、JAVA_HOME，以及 shell 配置里手写的 Node 路径。"
        ) +
        "</div></div>" +
        '<button class="btn-primary lg" id="btn-fix">立即修复</button>' +
        "</div>";
      return;
    }

    var item = selectedItem();
    if (!item) {
      content.innerHTML =
        (state.runtimes || []).length
          ? '<div class="empty">从左侧选择一个版本，或点右上角「重新扫描」</div>'
          : '<div class="empty">安装后点重新扫描，或导入安装目录</div>';
      return;
    }
    var active = isActive(item);
    content.innerHTML =
      '<div class="detail">' +
      "<h2>" +
      logoHtml(item.tool) +
      " " +
      esc(item.label || item.version) +
      (active ? ' <span class="tag success">激活</span>' : "") +
      "</h2>" +
      '<div class="field"><div class="field-label">版本</div><div class="field-value">' +
      esc(item.version) +
      "</div></div>" +
      '<div class="field"><div class="field-label">安装目录</div><div class="field-value"><code>' +
      esc(item.home) +
      "</code></div></div>" +
      '<div class="field"><div class="field-label">可执行文件</div><div class="field-value"><code>' +
      esc(item.binary) +
      "</code></div></div>" +
      '<div class="field"><div class="field-label">来源</div><div class="field-value">' +
      esc(sourceText(item.source)) +
      "</div></div>" +
      (active
        ? '<span class="dim">当前终端的 ' +
          ({ node: "node / npm", java: "java / javac", maven: "mvn", gradle: "gradle" }[item.tool] || item.tool) +
          " 已指向这个版本。</span>"
        : '<button class="btn-primary lg" data-use="' + esc(item.home) + '">切换到此版本</button>') +
      "</div>";
  }

  function renderStatus() {
    var paths = state.paths || {};
    document.getElementById("status-path").textContent = paths.localBin || "shim 目录";
    var parts = ["node", "java", "maven", "gradle"].map(function (tool) {
      var runtime = currentOf(tool);
      return tool + " " + (runtime ? runtime.version : "—");
    });
    document.getElementById("status-mid").textContent = parts.join(" · ");
    document.getElementById("status-ver").textContent = state.version ? "v" + state.version : "";
    var aboutVer = document.getElementById("about-ver");
    if (aboutVer) aboutVer.textContent = state.version ? "v" + state.version : "";
  }

  function render() {
    renderTabs();
    renderSider();
    renderContent();
    renderStatus();
    document.title = "DevSwitch";
    if (state.flash && state.flash.text) {
      toast(state.flash.text, state.flash.kind);
    }
  }

  window.__setState = function (next) {
    state = next || state;
    render();
  };

  document.getElementById("tabs").addEventListener("click", function (ev) {
    var tab = ev.target.closest("[data-page]");
    if (tab) send({ op: "page", page: tab.getAttribute("data-page") });
  });

  document.getElementById("btn-scan").addEventListener("click", function () {
    send({ op: "scan" });
  });
  document.getElementById("btn-import").addEventListener("click", function () {
    send({ op: "import" });
  });
  document.getElementById("btn-doctor").addEventListener("click", function () {
    send({ op: "page", page: "doctor" });
  });
  document.getElementById("btn-about").addEventListener("click", function () {
    document.getElementById("about").hidden = false;
  });
  document.getElementById("btn-about-close").addEventListener("click", function () {
    document.getElementById("about").hidden = true;
  });

  document.getElementById("sider").addEventListener("click", function (ev) {
    var use = ev.target.getAttribute("data-use");
    if (use) {
      send({ op: "use", home: use });
      return;
    }
    var card = ev.target.closest("[data-select]");
    if (card) send({ op: "select", home: card.getAttribute("data-select") });
    if (ev.target.id === "empty-scan") send({ op: "scan" });
    if (ev.target.id === "empty-import") send({ op: "import" });
  });

  document.getElementById("content").addEventListener("click", function (ev) {
    var use = ev.target.getAttribute("data-use");
    if (use) send({ op: "use", home: use });
    if (ev.target.id === "btn-fix") send({ op: "fix" });
  });
})();
