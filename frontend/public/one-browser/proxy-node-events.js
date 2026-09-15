(function (root) {
  function bind(container, handler) {
    if (!container || typeof container.addEventListener !== 'function') {
      throw new Error('代理节点容器不可用');
    }
    const onClick = function (event) {
      const card = event.target && event.target.closest
        ? event.target.closest('[data-proxy-id]')
        : null;
      if (!card) return;
      event.preventDefault();
      event.stopPropagation();
      const action = event.target.closest('[data-proxy-action]');
      return handler(
        (action && action.dataset.proxyId) || card.dataset.proxyId,
        (action && action.dataset.proxyAction) || 'select',
      );
    };
    container.addEventListener('click', onClick);
    return function unbind() {
      container.removeEventListener('click', onClick);
    };
  }

  function singleFlightDelete(handler) {
    const deleting = new Set();
    return function dispatch(id, action) {
      if (action !== 'delete') return handler(id, action);
      if (deleting.has(id)) return undefined;
      deleting.add(id);
      return Promise.resolve(handler(id, action)).finally(function () {
        deleting.delete(id);
      });
    };
  }

  const api = { bind, singleFlightDelete };
  root.OneBrowserProxyNodeEvents = api;
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
})(typeof globalThis !== 'undefined' ? globalThis : window);
