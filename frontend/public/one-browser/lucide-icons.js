/* Local Lucide icon subset (ISC). Kept in the portable bundle so icons never
   depend on a CDN or the user's network state. */
(() => {
  const icons = {
    'panel-top': '<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/>',
    'panels-top-left': '<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/>',
    cpu: '<rect width="16" height="16" x="4" y="4" rx="2"/><rect width="6" height="6" x="9" y="9" rx="1"/><path d="M9 1v3M15 1v3M9 20v3M15 20v3M20 9h3M20 14h3M1 9h3M1 14h3"/>',
    globe: '<circle cx="12" cy="12" r="10"/><path d="M2 12h20M12 2a15.3 15.3 0 0 1 0 20M12 2a15.3 15.3 0 0 0 0 20"/>',
    'circle-user-round': '<path d="M18 20a6 6 0 0 0-12 0"/><circle cx="12" cy="10" r="4"/><circle cx="12" cy="12" r="10"/>',
    blocks: '<rect width="7" height="7" x="14" y="3" rx="1"/><path d="M10 21V8a2 2 0 0 0-2-2H3M20 14v7M14 17h7"/><rect width="7" height="7" x="3" y="14" rx="1"/>',
    'list-filter': '<path d="M3 6h18M7 12h10M10 18h4"/>',
    'settings-2': '<path d="M20 7h-9M14 17H5"/><circle cx="17" cy="17" r="3"/><circle cx="7" cy="7" r="3"/>',
    'circle-help': '<circle cx="12" cy="12" r="10"/><path d="M9.1 9a3 3 0 1 1 5.8 1c0 2-3 2-3 4M12 18h.01"/>',
    'badge-help': '<path d="M3.85 8.62a4 4 0 0 1 4.78-4.77 4 4 0 0 1 6.74 0 4 4 0 0 1 4.78 4.78 4 4 0 0 1 0 6.74 4 4 0 0 1-4.77 4.78 4 4 0 0 1-6.75 0 4 4 0 0 1-4.78-4.77 4 4 0 0 1 0-6.76Z"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><path d="M12 17h.01"/>',
    'chevron-left': '<path d="m15 18-6-6 6-6"/>',
    upload: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M17 8l-5-5-5 5M12 3v12"/>',
    'cloud-upload': '<path d="M12 13v8M8 17l4-4 4 4"/><path d="M4.4 15.5A5 5 0 0 1 6 6.1 7 7 0 0 1 19.7 8.2 4.5 4.5 0 0 1 19.5 17H18"/>',
    plus: '<path d="M5 12h14M12 5v14"/>',
    search: '<circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/>',
    'refresh-cw': '<path d="M20 11a8.1 8.1 0 0 0-15.5-2M4 4v5h5M4 13a8.1 8.1 0 0 0 15.5 2M20 20v-5h-5"/>',
    check: '<path d="m20 6-11 11-5-5"/>',
    'arrow-up': '<path d="m18 15-6-6-6 6M12 9v12"/>',
    x: '<path d="M18 6 6 18M6 6l12 12"/>',
    ellipsis: '<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>',
    eye: '<path d="M2.1 12a10 10 0 0 1 19.8 0 10 10 0 0 1-19.8 0"/><circle cx="12" cy="12" r="3"/>',
    'chevron-down': '<path d="m6 9 6 6 6-6"/>',
    'edit-3': '<path d="M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L8 18l-4 1 1-4Z"/>',
    copy: '<rect width="14" height="14" x="8" y="8" rx="2"/><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>',
    'trash-2': '<path d="M3 6h18M8 6V4h8v2M19 6l-1 15H6L5 6M10 11v6M14 11v6"/>',
    'log-in': '<path d="M10 17l5-5-5-5M15 12H3M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/>',
    repeat: '<path d="m17 1 4 4-4 4M3 11V9a4 4 0 0 1 4-4h14M7 23l-4-4 4-4M21 13v2a4 4 0 0 1-4 4H3"/>',
    'log-out': '<path d="m10 17 5-5-5-5M15 12H3M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4"/>',
    'user-round-plus': '<path d="M2 21a8 8 0 0 1 13.3-6M10 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8M19 8v6M22 11h-6"/>',
    'monitor-cog': '<path d="M12 17v4M8 21h8M3 13V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v8"/><circle cx="18" cy="17" r="3"/><path d="M18 12v2M18 20v2M13 17h2M21 17h2"/>',
    'map-pin': '<path d="M20 10c0 5-8 12-8 12S4 15 4 10a8 8 0 1 1 16 0"/><circle cx="12" cy="10" r="3"/>',
    network: '<rect x="9" y="2" width="6" height="6" rx="1"/><rect x="3" y="16" width="6" height="6" rx="1"/><rect x="15" y="16" width="6" height="6" rx="1"/><path d="M12 8v4M6 16v-2h12v2"/>',
    package: '<path d="m7.5 4.3 9 5.2M3.3 7l8.7 5 8.7-5M12 22V12"/><path d="M21 16V8a2 2 0 0 0-1-1.7l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.7l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16"/>',
    'folder-open': '<path d="m6 14 1.5-3h12l-2 8H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4l2 3h6a2 2 0 0 1 2 2v1"/>',
    download: '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/>',
    'hard-drive-download': '<path d="M12 2v8M9 7l3 3 3-3"/><rect width="20" height="8" x="2" y="14" rx="2"/><path d="M6 18h.01M10 18h.01"/>',
    'shield-check': '<path d="M20 13c0 5-3.5 7.5-8 9-4.5-1.5-8-4-8-9V5l8-3 8 3Z"/><path d="m9 12 2 2 4-4"/>',
    'circle-alert': '<circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/>',
    eraser: '<path d="m7 21-4.3-4.3c-1-1-1-2.5 0-3.4l9.6-9.6c1-1 2.5-1 3.4 0l5.6 5.6c1 1 1 2.5 0 3.4L13 21"/><path d="M22 21H7"/><path d="m5 11 9 9"/>',
    'file-input': '<path d="M4 22h14a2 2 0 0 0 2-2V7.5L14.5 2H6a2 2 0 0 0-2 2v4"/><path d="M14 2v6h6"/><path d="M2 15h10"/><path d="m9 18 3-3-3-3"/>',
    'file-output': '<path d="M4 22h14a2 2 0 0 0 2-2V7.5L14.5 2H6a2 2 0 0 0-2 2v4"/><path d="M14 2v6h6"/><path d="M2 15h10"/><path d="m5 12-3 3 3 3"/>',
    'database-zap': '<ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M3 5v14c0 1.66 4.03 3 9 3 1.08 0 2.12-.06 3.05-.18"/><path d="M21 5v3"/><path d="m21 12-3 5h4l-3 5"/><path d="M3 12c0 1.66 4.03 3 9 3 .91 0 1.78-.05 2.59-.13"/>',
  };
  function svg(name, className = '') {
    const body = icons[name] || icons['circle-help'];
    return `<svg class="lucide lucide-${name}${className ? ` ${className}` : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">${body}</svg>`;
  }
  function hydrate(root = document) {
    root.querySelectorAll('[data-lucide]').forEach(node => {
      if (node.dataset.lucideReady === '1') return;
      node.innerHTML = svg(node.dataset.lucide);
      node.dataset.lucideReady = '1';
    });
    root.querySelectorAll('button[aria-label="关闭"], .modal-close').forEach(node => { node.innerHTML = svg('x'); });
    root.querySelectorAll('[data-password-toggle]').forEach(node => { node.innerHTML = svg('eye'); });
    const accountMenu = root.querySelector('#accountMenuBtn');
    if (accountMenu) accountMenu.innerHTML = svg('ellipsis');
    root.querySelectorAll('.confirm-delete-icon').forEach(node => { node.innerHTML = svg('circle-alert'); });
    root.querySelectorAll('.notice>span').forEach(node => { node.innerHTML = svg('shield-check'); });
  }
  window.OneBrowserIcons = { svg, hydrate };
})();
