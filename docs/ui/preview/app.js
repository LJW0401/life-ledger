// Static UI preview behavior for life-ledger; replace with React state and API calls in the product app.
const routes = {
  "/": "important-dates",
  "/important-dates": "important-dates",
  "/transactions": "transactions",
  "/decisions": "decisions",
};

const state = {
  view: {
    "important-dates": "upcoming",
    transactions: "list",
    decisions: "active",
  },
};

const pageMeta = {
  "important-dates": {
    title: "日期",
    heading: "重要日期不再散落在聊天和备忘录里",
    intro: "用一个聚焦列表管理生日、纪念日、证件到期和缴费日。首屏优先展示即将发生的事项。",
  },
  transactions: {
    title: "账单",
    heading: "收支流水、预算和 Excel 交换放在同一个工作台",
    intro: "账单页默认展示流水，侧边保留月度概览和预算使用情况。预算按月份和分类管理，不单独占用一级页面。",
  },
  decisions: {
    title: "决策",
    heading: "把关键选择从一时想法沉淀成可复盘记录",
    intro: "决策页按状态组织：进行中、待复盘、已归档。每条记录保留背景、方案和下一次复盘时间。",
  },
};

const dates = [
  { day: "03", month: "8月", title: "护照到期", detail: "记录换证材料和办理进度", tags: ["证件", "到期"], tone: "green" },
  { day: "12", month: "8月", title: "家庭年度体检", detail: "父母体检套餐续约，记录预约信息", tags: ["健康", "重复每年"], tone: "blue" },
  { day: "26", month: "8月", title: "房租付款日", detail: "每月 26 日固定日期，账单页可关联支出", tags: ["缴费", "每月"], tone: "amber" },
  { day: "09", month: "9月", title: "结婚纪念日", detail: "记录礼物、安排和当日计划", tags: ["纪念日", "重要"], tone: "red" },
];

const transactions = [
  { date: "2026-07-04", time: "12:20", type: "支出", category: "餐饮", book: "默认账本", budget: "是", account: "支付宝", merchant: "午餐", amount: "-38.00" },
  { date: "2026-07-03", time: "09:10", type: "收入", category: "工资", book: "默认账本", budget: "否", account: "招商银行", merchant: "薪资", amount: "+18500.00" },
  { date: "2026-07-02", time: "08:42", type: "支出", category: "交通", book: "默认账本", budget: "是", account: "微信", merchant: "地铁", amount: "-7.00" },
  { date: "2026-07-01", time: "10:00", type: "支出", category: "居住", book: "默认账本", budget: "是", account: "招商银行", merchant: "房租", amount: "-4200.00" },
  { date: "2026-06-30", time: "21:16", type: "支出", category: "学习", book: "默认账本", budget: "是", account: "信用卡", merchant: "课程订阅", amount: "-299.00" },
];

const decisions = {
  active: [
    { title: "是否更换主力记账方式", detail: "对比当前表格、手机 App 和 life-ledger 的长期维护成本。", due: "7月12日", owner: "个人" },
    { title: "今年是否配置 NAS 备份", detail: "关注照片、账单附件和重要文档的本地备份可靠性。", due: "7月20日", owner: "家庭" },
  ],
  review: [
    { title: "通勤方式调整", detail: "上月改为地铁加步行，复盘时间成本和体力消耗。", due: "今天", owner: "个人" },
  ],
  archived: [
    { title: "年度体检套餐选择", detail: "选择家庭套餐，已记录价格、项目和实际体验。", due: "已复盘", owner: "家庭" },
  ],
};

const app = document.querySelector("#app");
const pageTitle = document.querySelector("#page-title");

function normalizePath(pathname) {
  return routes[pathname] ? pathname : "/important-dates";
}

function currentPage() {
  return routes[normalizePath(window.location.pathname)];
}

function navigate(path) {
  const nextPath = normalizePath(path);
  if (window.location.pathname !== nextPath) {
    window.history.pushState({}, "", nextPath);
  }
  render();
}

function setView(page, view) {
  state.view[page] = view;
  render();
}

function intro(page) {
  const meta = pageMeta[page];
  return `
    <div class="page-intro">
      <div>
        <h2>${meta.heading}</h2>
        <p>${meta.intro}</p>
      </div>
      <div class="action-row">
        ${page === "transactions" ? '<button class="secondary-button" type="button">导入</button><button class="secondary-button" type="button">导出</button>' : ""}
        <button class="primary-button" type="button">新增</button>
      </div>
    </div>
  `;
}

function segmented(page, items) {
  const active = state.view[page];
  return `
    <div class="segmented" role="tablist">
      ${items.map((item) => `
        <button type="button" class="${active === item.key ? "active" : ""}" data-view="${item.key}">
          ${item.label}
        </button>
      `).join("")}
    </div>
  `;
}

function metrics(items) {
  return `
    <div class="metric-row">
      ${items.map((item) => `
        <div class="metric">
          <span>${item.label}</span>
          <strong>${item.value}</strong>
        </div>
      `).join("")}
    </div>
  `;
}

function renderDates() {
  return `
    ${intro("important-dates")}
    ${segmented("important-dates", [
      { key: "upcoming", label: "即将到来" },
      { key: "all", label: "全部日期" },
      { key: "calendar", label: "日历视图" },
    ])}
    <div class="content-grid">
      <section class="panel wide">
        <div class="section-title">
          <h3>未来 60 天</h3>
          <span>按距离今天排序</span>
        </div>
        <div class="timeline">
          ${dates.map((item) => `
            <article class="list-item">
              <div class="date-badge">${item.day}<small>${item.month}</small></div>
              <div class="item-main">
                <h4>${item.title}</h4>
                <p>${item.detail}</p>
              </div>
              <div class="tag-row">
                ${item.tags.map((tag) => `<span class="tag ${item.tone}">${tag}</span>`).join("")}
              </div>
            </article>
          `).join("")}
        </div>
      </section>
      <aside class="panel side">
        <div class="section-title">
          <h3>日期概览</h3>
          <span>首版指标</span>
        </div>
        ${metrics([
          { label: "30 天内", value: "3" },
          { label: "重复日期", value: "9" },
          { label: "证件日期", value: "2" },
        ])}
        <div class="section-title" style="margin-top: 18px;">
          <h3>本周日历</h3>
          <span>轻量提示</span>
        </div>
        <div class="calendar-strip">
          ${["一", "二", "三", "四", "五", "六", "日"].map((day, index) => `
            <div class="day-cell ${index === 5 ? "marked" : ""}">
              <strong>周${day}</strong>
              <span>${index === 5 ? "房租付款" : "无事项"}</span>
            </div>
          `).join("")}
        </div>
      </aside>
    </div>
  `;
}

function renderTransactions() {
  return `
    ${intro("transactions")}
    ${segmented("transactions", [
      { key: "list", label: "流水列表" },
      { key: "stats", label: "统计概览" },
      { key: "budget", label: "预算概览" },
      { key: "exchange", label: "导入导出" },
    ])}
    <div class="content-grid">
      <section class="panel wide">
        <div class="section-title">
          <h3>7 月流水</h3>
          <span>5 条示例记录</span>
        </div>
        <div class="table-shell">
          <table>
            <thead>
              <tr>
                <th>日期</th>
                <th>时间</th>
                <th>类型</th>
                <th>分类</th>
                <th>计入预算</th>
                <th>所属账本</th>
                <th>账户</th>
                <th>对象</th>
                <th>金额</th>
              </tr>
            </thead>
            <tbody>
              ${transactions.map((item) => `
                <tr>
                  <td>${item.date}</td>
                  <td>${item.time}</td>
                  <td><span class="tag ${item.type === "收入" ? "green" : "red"}">${item.type}</span></td>
                  <td>${item.category}</td>
                  <td>${item.budget}</td>
                  <td>${item.book}</td>
                  <td>${item.account}</td>
                  <td>${item.merchant}</td>
                  <td class="amount ${item.type === "收入" ? "income" : ""}">${item.amount}</td>
                </tr>
              `).join("")}
            </tbody>
          </table>
        </div>
      </section>
      <aside class="panel side">
        <div class="section-title">
          <h3>月度概览</h3>
          <span>示例数据</span>
        </div>
        ${metrics([
          { label: "收入", value: "18,500" },
          { label: "支出", value: "4,544" },
          { label: "结余", value: "13,956" },
        ])}
        <div class="section-title" style="margin-top: 18px;">
          <h3>预算使用</h3>
          <span>7 月</span>
        </div>
        <div class="bar-list">
          ${[
            ["居住", 4200, 4500],
            ["学习", 299, 500],
            ["餐饮", 38, 1800],
          ].map(([name, used, total]) => {
            const width = Math.min(Math.round((used / total) * 100), 100);
            return `
              <div class="bar-row">
                <header><span>${name}</span><strong>${used}/${total}</strong></header>
                <div class="bar-track"><div class="bar-fill" style="width: ${width}%;"></div></div>
              </div>
            `;
          }).join("")}
        </div>
        <div class="section-title" style="margin-top: 18px;">
          <h3>支出分类</h3>
          <span>占比</span>
        </div>
        <div class="bar-list">
          ${[
            ["居住", 82],
            ["学习", 8],
            ["餐饮", 7],
            ["交通", 3],
          ].map(([name, width]) => `
            <div class="bar-row">
              <header><span>${name}</span><strong>${width}%</strong></header>
              <div class="bar-track"><div class="bar-fill" style="width: ${width}%;"></div></div>
            </div>
          `).join("")}
        </div>
      </aside>
      <section class="panel full">
        <div class="section-title">
          <h3>Excel 交换</h3>
          <span>导入前整表校验，失败不入库</span>
        </div>
        <p class="empty-note">这里预留模板下载、上传 .xlsx、导入错误明细和导出当前筛选结果的入口。</p>
      </section>
    </div>
  `;
}

function renderDecisions() {
  const lanes = [
    { key: "active", label: "进行中", tone: "blue" },
    { key: "review", label: "待复盘", tone: "amber" },
    { key: "archived", label: "已归档", tone: "green" },
  ];

  return `
    ${intro("decisions")}
    ${segmented("decisions", lanes.map((lane) => ({ key: lane.key, label: lane.label })))}
    <div class="content-grid">
      <section class="panel full">
        <div class="section-title">
          <h3>决策看板</h3>
          <span>按状态组织</span>
        </div>
        <div class="decision-board">
          ${lanes.map((lane) => `
            <div class="decision-lane">
              <h3 class="lane-title">${lane.label}</h3>
              ${decisions[lane.key].map((item) => `
                <article class="decision-card">
                  <div class="tag-row"><span class="tag ${lane.tone}">${lane.label}</span></div>
                  <h4 style="margin-top: 10px;">${item.title}</h4>
                  <p>${item.detail}</p>
                  <div class="decision-meta">
                    <span>${item.owner}</span>
                    <strong>${item.due}</strong>
                  </div>
                </article>
              `).join("")}
            </div>
          `).join("")}
        </div>
      </section>
    </div>
  `;
}

function render() {
  const page = currentPage();
  pageTitle.textContent = pageMeta[page].title;

  document.querySelectorAll("[data-route]").forEach((button) => {
    button.classList.toggle("active", normalizePath(button.dataset.route) === normalizePath(window.location.pathname));
  });

  if (page === "important-dates") app.innerHTML = renderDates();
  if (page === "transactions") app.innerHTML = renderTransactions();
  if (page === "decisions") app.innerHTML = renderDecisions();
}

document.addEventListener("click", (event) => {
  const routeButton = event.target.closest("[data-route]");
  if (routeButton) {
    navigate(routeButton.dataset.route);
    return;
  }

  const viewButton = event.target.closest("[data-view]");
  if (viewButton) {
    setView(currentPage(), viewButton.dataset.view);
  }
});

window.addEventListener("popstate", render);

if (!routes[window.location.pathname]) {
  window.history.replaceState({}, "", "/important-dates");
}

render();
