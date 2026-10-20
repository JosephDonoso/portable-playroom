'use strict';
const address = document.getElementById('network-url');
const status = document.getElementById('copy-status');
const copy = document.getElementById('copy');
fetch('/api/info').then(response => {
  if (!response.ok) throw new Error('No disponible');
  return response.json();
}).then(info => {
  const url = info.addresses[0];
  document.getElementById('network-title').textContent = url ? 'Tu sala está en la red local' : 'Menú disponible en este PC';
  address.textContent = url || location.origin;
  copy.disabled = false;
}).catch(() => {
  document.getElementById('network-title').textContent = 'No se pudo consultar la red local';
  address.textContent = location.origin;
  copy.disabled = false;
});
copy.addEventListener('click', async () => {
  try {
    if (!navigator.clipboard?.writeText) throw new Error('Copia manual');
    await navigator.clipboard.writeText(address.textContent);
    status.textContent = 'Enlace copiado.';
  } catch {
    const range = document.createRange();
    range.selectNodeContents(address);
    const selection = window.getSelection();
    selection.removeAllRanges();
    selection.addRange(range);
    status.textContent = 'Enlace seleccionado. Mantén pulsado o usa Ctrl+C para copiarlo.';
  }
});
