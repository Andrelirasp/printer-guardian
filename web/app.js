const input = document.querySelector('#log-input');
const fileInput = document.querySelector('#file-input');
const dropZone = document.querySelector('#drop-zone');
const results = document.querySelector('#results');
const emptyState = document.querySelector('#empty-state');
const eventsElement = document.querySelector('#events');
let analysis = null;
let activeFilter = 'all';

const rules = [
  { type: 'error', match: /\b(erro|error|falha|failed|fatal|panic|timeout|não encontrado|nao encontrado|sem porta|não corrigida|nao corrigida)\b/i },
  { type: 'warning', match: /\b(alerta|offline|unknown|paused|desconect|warning|warn|não foi possível|nao foi possivel|não conseguiu|nao conseguiu)\b/i },
  { type: 'success', match: /\b(corrigid|fix(ed)?|reativad|reiniciado|restaurad|reaberto|iniciado|inicializado|running|configuração carregada|configuracao carregada)\b/i },
];

function classify(message) {
  const rule = rules.find(({ match }) => match.test(message));
  return rule ? rule.type : 'info';
}

function parseLine(line, index) {
  const match = line.match(/^(\d{4}[/-]\d{2}[/-]\d{2}[ T]\d{2}:\d{2}:\d{2})\s+(.*)$/);
  const timestamp = match ? match[1] : '';
  const message = (match ? match[2] : line).trim();
  return { index, timestamp, message, type: classify(message), isQZ: /\bqz\s*tray\b|\bqz[_ -]?|qz-tray|javaw/i.test(message) };
}

function analyse(text) {
  const lines = text.split(/\r?\n/).map((line, index) => parseLine(line, index + 1)).filter(({ message }) => message);
  const relevant = lines.filter(({ type, isQZ }) => type !== 'info' || isQZ);
  const counts = ['error', 'warning', 'success'].reduce((result, type) => ({ ...result, [type]: lines.filter((event) => event.type === type).length }), {});
  const qzEvents = lines.filter(({ isQZ }) => isQZ);
  const timestamps = lines.filter(({ timestamp }) => timestamp).map(({ timestamp }) => timestamp);
  return { lines, relevant, counts, qzEvents, timestamps };
}

function getFindings(data) {
  const findings = [];
  const messages = data.lines.map(({ message }) => message).join('\n');
  const qzRunning = /QZ_RUNNING|QZ Tray já está rodando|QZ Tray ja esta rodando/i.test(messages);
  const qzStarted = /QZ_STARTED|QZ Tray reiniciado|QZ Tray reaberto/i.test(messages);
  const qzMissing = /QZ_NOT_FOUND|QZ Tray não encontrado|QZ Tray nao encontrado/i.test(messages);
  const qzError = /QZ.*(erro|error|falha|failed|timeout)|Erro ao verificar QZ/i.test(messages);

  if (qzMissing) findings.push({ type: 'error', title: 'QZ Tray não foi localizado', text: 'O agente não encontrou o executável nos caminhos configurados. Confirme a instalação do QZ Tray no cliente.' });
  if (qzError) findings.push({ type: 'error', title: 'Falha ao monitorar o QZ Tray', text: 'O PowerShell retornou erro durante a detecção ou abertura do QZ Tray. Veja os eventos destacados abaixo.' });
  if (qzRunning) findings.push({ type: 'success', title: 'QZ Tray está em execução', text: 'O log registrou um processo do QZ Tray ou do runtime Java associado a ele.' });
  if (qzStarted) findings.push({ type: 'success', title: 'QZ Tray foi iniciado pelo agente', text: 'O Printer Guardian detectou o QZ fechado e tentou reabri-lo.' });
  if (/PowerShell timeout/i.test(messages)) findings.push({ type: 'error', title: 'PowerShell excedeu o tempo limite', text: 'Uma rotina do Windows levou mais de 60 segundos e foi encerrada.' });
  if (/USB_NOT_FIXED|BT_NOT_FIXED|NOT_FIXED/i.test(messages)) findings.push({ type: 'warning', title: 'Impressora sem correção automática', text: 'O agente detectou uma impressora problemática, mas não conseguiu aplicar a correção.' });
  if (/SNMP_FIXED/i.test(messages)) findings.push({ type: 'success', title: 'SNMP corrigido', text: 'O agente desativou SNMP em uma ou mais portas de rede para evitar falso offline.' });
  if (/USB_FIXED|BT_FIXED/i.test(messages)) findings.push({ type: 'success', title: 'Porta de impressora corrigida', text: 'O agente aplicou uma correção de porta USB ou Bluetooth.' });
  if (data.counts.error && !findings.some(({ type }) => type === 'error')) findings.push({ type: 'error', title: `${data.counts.error} erro(s) detectado(s)`, text: 'Revise os eventos em vermelho para encontrar a causa específica.' });
  if (data.counts.warning && !findings.some(({ type }) => type === 'warning')) findings.push({ type: 'warning', title: `${data.counts.warning} alerta(s) detectado(s)`, text: 'Há mensagens que merecem acompanhamento, mesmo sem uma falha definitiva.' });
  if (!findings.length) findings.push({ type: 'success', title: 'Nenhuma ocorrência crítica encontrada', text: 'O conteúdo não contém erros, alertas ou eventos conhecidos do Printer Guardian.' });
  return findings;
}

function renderEvent(event) {
  const node = document.createElement('article');
  node.className = `event ${event.type}`;
  node.dataset.type = event.type;
  const tag = document.createElement('span');
  tag.className = `tag ${event.type}`;
  tag.textContent = { error: 'ERRO', warning: 'ALERTA', success: 'OK', info: 'INFO' }[event.type];
  const content = document.createElement('div');
  if (event.timestamp) {
    const time = document.createElement('span');
    time.className = 'event-time';
    time.textContent = event.timestamp;
    content.append(time);
  }
  const message = document.createElement('div');
  message.className = 'event-message';
  message.textContent = event.message;
  content.append(message);
  node.append(tag, content);
  return node;
}

function renderEvents() {
  eventsElement.replaceChildren();
  const list = analysis.relevant.filter((event) => activeFilter === 'all' || event.type === activeFilter);
  if (!list.length) {
    eventsElement.textContent = 'Nenhum evento corresponde ao filtro selecionado.';
    return;
  }
  list.slice(-300).reverse().forEach((event) => eventsElement.append(renderEvent(event)));
}

function renderQZ(events) {
  const summary = document.querySelector('#qz-summary');
  const badge = document.querySelector('#qz-badge');
  const container = document.querySelector('#qz-events');
  container.replaceChildren();
  const messages = events.map(({ message }) => message).join('\n');
  let state = 'neutral';
  let label = 'Sem dados';
  if (/QZ_NOT_FOUND|QZ Tray não encontrado|QZ Tray nao encontrado|Erro ao verificar QZ/i.test(messages)) { state = 'error'; label = 'Com falha'; }
  else if (/QZ_RUNNING|já está rodando|ja esta rodando/i.test(messages)) { state = 'success'; label = 'Em execução'; }
  else if (/QZ_STARTED|reiniciado|reaberto/i.test(messages)) { state = 'warning'; label = 'Reiniciado'; }
  badge.className = `badge ${state}`;
  badge.textContent = label;
  summary.textContent = events.length ? `${events.length} evento(s) relacionado(s) ao QZ Tray encontrado(s).` : 'O log não possui mensagens relacionadas ao QZ Tray.';
  events.slice(-8).reverse().forEach((event) => container.append(renderEvent(event)));
}

function render() {
  const data = analysis;
  const findings = getFindings(data);
  document.querySelector('#total-lines').textContent = data.lines.length.toLocaleString('pt-BR');
  document.querySelector('#error-count').textContent = data.counts.error;
  document.querySelector('#warning-count').textContent = data.counts.warning;
  document.querySelector('#fix-count').textContent = data.counts.success;
  const range = document.querySelector('#range');
  range.textContent = data.timestamps.length ? `${data.timestamps[0]} — ${data.timestamps.at(-1)}` : 'Sem timestamp';
  const verdict = document.querySelector('#verdict');
  const highest = data.counts.error ? 'error' : data.counts.warning ? 'warning' : 'success';
  verdict.className = `verdict ${highest}`;
  verdict.textContent = data.counts.error ? `Foram encontrados ${data.counts.error} erro(s). Priorize os eventos críticos antes de qualquer nova ação no cliente.` : data.counts.warning ? `Nenhum erro crítico, mas há ${data.counts.warning} alerta(s) que precisam de acompanhamento.` : 'Nenhum erro ou alerta conhecido foi identificado no conteúdo analisado.';
  const findingsElement = document.querySelector('#findings');
  findingsElement.replaceChildren();
  findings.forEach((finding) => {
    const item = document.createElement('article');
    item.className = `finding ${finding.type}`;
    const title = document.createElement('strong');
    title.textContent = finding.title;
    const text = document.createElement('p');
    text.textContent = finding.text;
    item.append(title, text);
    findingsElement.append(item);
  });
  renderQZ(data.qzEvents);
  renderEvents();
  emptyState.hidden = true;
  results.hidden = false;
}

function runAnalysis() {
  const text = input.value.trim();
  if (!text) return;
  analysis = analyse(text);
  render();
}

function loadFile(file) {
  if (!file) return;
  const reader = new FileReader();
  reader.onload = () => { input.value = String(reader.result || ''); document.querySelector('#file-name').textContent = file.name; runAnalysis(); };
  reader.readAsText(file);
}

document.querySelector('#analyze').addEventListener('click', runAnalysis);
document.querySelector('#clear').addEventListener('click', () => {
  input.value = ''; fileInput.value = ''; analysis = null; results.hidden = true; emptyState.hidden = false; document.querySelector('#file-name').textContent = 'Nenhum arquivo selecionado';
});
fileInput.addEventListener('change', () => loadFile(fileInput.files[0]));
['dragenter', 'dragover'].forEach((type) => dropZone.addEventListener(type, (event) => { event.preventDefault(); dropZone.classList.add('dragging'); }));
['dragleave', 'drop'].forEach((type) => dropZone.addEventListener(type, (event) => { event.preventDefault(); dropZone.classList.remove('dragging'); }));
dropZone.addEventListener('drop', (event) => loadFile(event.dataTransfer.files[0]));
document.querySelectorAll('.filter').forEach((button) => button.addEventListener('click', () => {
  activeFilter = button.dataset.filter;
  document.querySelectorAll('.filter').forEach((item) => item.classList.toggle('active', item === button));
  if (analysis) renderEvents();
}));
