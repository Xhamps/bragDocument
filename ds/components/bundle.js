/* @ds-bundle: {"format":4,"namespace":"BragDocUI","components":[{"name":"Button"},{"name":"IconButton"},{"name":"ButtonGroup"},{"name":"Tooltip"},{"name":"SegmentedControl"},{"name":"SearchField"},{"name":"Card"},{"name":"MediaCard"},{"name":"HeroHeader"},{"name":"NotificationItem"},{"name":"MenuList"},{"name":"UserMenu"},{"name":"TopBar"},{"name":"PageHeader"},{"name":"Toggle"},{"name":"TextField"},{"name":"TextArea"},{"name":"Select"},{"name":"Checkbox"},{"name":"RadioGroup"},{"name":"Slider"},{"name":"Stepper"},{"name":"FieldTile"},{"name":"Tag"},{"name":"MediaCell"},{"name":"DataTable"},{"name":"Pagination"}]} */
(function () {
  var R = window.React, h = R.createElement;
  function cx() { return Array.prototype.filter.call(arguments, Boolean).join(' '); }
  function svg(d, extra) {
    return h('svg', Object.assign({ viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: 1.5, strokeLinecap: 'round', strokeLinejoin: 'round', 'aria-hidden': 'true' }, extra || {}), h('path', { d: d }));
  }
  var ICONS = {
    chevron: 'M9 6l6 6-6 6',
    search: 'M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM20 20l-4-4',
    check: 'M5 12.5l4.5 4.5L19 7.5',
    chevronDown: 'M6 9l6 6 6-6',
    plus: 'M12 6v12M6 12h12',
    minus: 'M6 12h12',
    mail: 'M4 6h16v12H4zM4 7l8 6 8-6',
    arrowLeft: 'M19 12H5M11 6l-6 6 6 6',
    arrowRight: 'M5 12h14M13 6l6 6-6 6',
    download: 'M12 4v11M7 10l5 5 5-5M5 20h14',
    card: 'M3 6h18v12H3zM3 10h18M7 15h4',
    plusCircle: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 8v8M8 12h8',
    moon: 'M20 14.5A8 8 0 0 1 9.5 4 8 8 0 1 0 20 14.5z',
    sun: 'M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4',
    play: 'M8 5v14l11-7z',
    home: 'M4 11l8-7 8 7M6 9.5V20h12V9.5M10 20v-5h4v5',
    bell: 'M6 16V11a6 6 0 1 1 12 0v5l2 2H4zM10 20a2 2 0 0 0 4 0',
    sliders: 'M4 7h10M18 7h2M4 17h4M12 17h8M16 5v4M10 15v4',
    eye: 'M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12zM12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6z',
    share: 'M12 3v12M7 8l5-5 5 5M5 14v6h14v-6',
    close: 'M6 6l12 12M18 6L6 18',
    external: 'M14 4h6v6M20 4l-9 9M18 14v6H4V6h6',
    menu: 'M4 7h16M4 12h16M4 17h16',
    user: 'M12 4a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM4 20c1.5-3.5 4.5-5 8-5s6.5 1.5 8 5',
    signOut: 'M10 5H5v14h5M14 8l4 4-4 4M18 12H9',
    settings: 'M12 9a3 3 0 1 0 0 6 3 3 0 0 0 0-6zM12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M5.6 18.4l2.1-2.1M16.3 7.7l2.1-2.1',
    grid: 'M5 5h5v5H5zM14 5h5v5h-5zM5 14h5v5H5zM14 14h5v5h-5z',
    book: 'M5 4h10a3 3 0 0 1 3 3v13H8a3 3 0 0 1-3-3zM5 17a3 3 0 0 1 3-3h10',
    chart: 'M5 19V10M12 19V5M19 19v-7',
    help: 'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM9.5 9.5a2.5 2.5 0 1 1 3.5 2.3c-.6.3-1 .8-1 1.4V14M12 17h.01',
    sort: 'M8 9l4-4 4 4M8 15l4 4 4-4',
    sortUp: 'M8 14l4-4 4 4',
    sortDown: 'M8 10l4 4 4-4',
    more: 'M4.75 12a1.25 1.25 0 1 0 2.5 0 1.25 1.25 0 1 0-2.5 0M10.75 12a1.25 1.25 0 1 0 2.5 0 1.25 1.25 0 1 0-2.5 0M16.75 12a1.25 1.25 0 1 0 2.5 0 1.25 1.25 0 1 0-2.5 0',
    edit: 'M4 20h4L19 9l-4-4L4 16zM14 6l4 4',
    trash: 'M5 7h14M10 7V4h4v3M7 7l1 13h8l1-13'
  };
  function Icon(props) { var x = { className: props.className, width: props.size || 20, height: props.size || 20 }; if (props.name === 'play' || props.name === 'more') { x.fill = 'currentColor'; x.stroke = 'none'; } return svg(ICONS[props.name] || props.name, x); }
  function iconNode(i, size) { return i == null ? null : (typeof i === 'string' ? h(Icon, { name: i, size: size }) : i); }

  function Button(p) {
    var variant = p.variant || 'glass', size = p.size || 'md';
    var rest = Object.assign({}, p); ['variant', 'size', 'chevron', 'glow', 'icon', 'trailingIcon', 'pill', 'fullWidth', 'className', 'children'].forEach(function (k) { delete rest[k]; });
    return h('button', Object.assign({ type: 'button' }, rest, {
      className: cx('bd-btn', 'bd-btn-' + variant, size !== 'md' && 'bd-btn-' + size, p.glow && 'bd-btn-glow', p.pill && 'bd-btn-pill', p.fullWidth && 'bd-btn-block', p.className)
    }), iconNode(p.icon, 16), p.children, iconNode(p.trailingIcon, 16), p.chevron ? h(Icon, { name: 'chevron', size: 16 }) : null);
  }

  function HeroHeader(p) {
    var size = p.size || 'display', align = p.align || 'center', grad = p.gradient || 'primary';
    var Tag = p.as || 'h1';
    return h('header', { className: cx('bd-hero', 'bd-hero-' + align, 'bd-hero-' + size, p.className) },
      p.eyebrow ? h('div', { className: 'bd-hero-eyebrow' }, typeof p.eyebrow === 'string' ? h(Button, { size: 'sm', chevron: !!p.onEyebrowClick, onClick: p.onEyebrowClick }, p.eyebrow) : p.eyebrow) : null,
      h(Tag, { className: 'bd-hero-title' },
        p.lead ? h('span', { className: 'bd-hero-lead' }, p.lead, ' ') : null,
        h('span', { className: grad === 'none' ? null : 'bd-text-gradient' + (grad === 'primary' ? '' : '-' + grad) }, p.title)),
      p.subtitle ? h('p', { className: 'bd-hero-sub' }, p.subtitle) : null,
      p.actions ? h('div', { className: 'bd-hero-actions' }, p.actions) : null);
  }

  function useIndicator(value, deps, axis) {
    var box = R.useRef(null), st = R.useState(null), pos = st[0], setPos = st[1];
    var ready = R.useRef(false);
    function measure() {
      var el = box.current && box.current.querySelector('[data-bd-on="true"]');
      if (!el) { setPos(null); return; }
      var b = box.current.getBoundingClientRect(), r = el.getBoundingClientRect();
      setPos({ x: r.left - b.left - box.current.clientLeft, y: r.top - b.top - box.current.clientTop, w: r.width, h: r.height });
    }
    (R.useLayoutEffect || R.useEffect)(function () { measure(); }, [value].concat(deps || []));
    R.useEffect(function () {
      var raf = requestAnimationFrame(function () { ready.current = true; });
      window.addEventListener('resize', measure);
      var ro = window.ResizeObserver ? new ResizeObserver(function () { measure(); }) : null;
      if (ro && box.current) ro.observe(box.current);
      if (document.fonts && document.fonts.ready) document.fonts.ready.then(measure);
      return function () { cancelAnimationFrame(raf); window.removeEventListener('resize', measure); if (ro) ro.disconnect(); };
    }, []);
    var style = pos ? { width: pos.w + 'px', height: pos.h + 'px', transform: 'translate(' + pos.x + 'px, ' + pos.y + 'px)', opacity: 1 } : { opacity: 0 };
    return { ref: box, style: style, animated: ready.current };
  }

  function MediaCard(p) {
    var layout = p.layout || 'vertical';
    var img = p.image, media;
    var ratio = p.aspect || (layout === 'horizontal' ? '1 / 1' : layout === 'overlay' ? '4 / 5' : '16 / 9');
    if (img && /^gradient-/.test(img)) media = h('span', { className: 'bd-mcard-img', style: { background: 'var(--' + img + ')' }, role: p.imageAlt ? 'img' : undefined, 'aria-label': p.imageAlt });
    else if (img) media = h('img', { className: 'bd-mcard-img', src: img, alt: p.imageAlt || '', loading: 'lazy' });
    else media = h('span', { className: 'bd-mcard-img bd-mcard-empty', 'aria-hidden': 'true' });
    var author = p.author ? h('div', { className: 'bd-mcard-author' },
      p.author.avatar ? h('img', { className: 'bd-avatar bd-mcard-av', src: p.author.avatar, alt: '' }) : h('span', { className: 'bd-avatar bd-mcard-av', 'aria-hidden': 'true' }, initials(p.author.name)),
      h('span', { className: 'bd-mcard-author-text' }, h('span', { className: 'bd-mcard-author-name' }, p.author.name), p.author.role ? h('span', { className: 'bd-mcard-author-role' }, p.author.role) : null)) : null;
    var progress = p.progress != null ? h('div', { className: 'bd-mcard-progress', role: 'progressbar', 'aria-valuemin': 0, 'aria-valuemax': 100, 'aria-valuenow': p.progress, 'aria-label': p.progressLabel || 'Progress' },
      h('span', { style: { width: Math.max(0, Math.min(100, p.progress)) + '%' } })) : null;
    var Title = p.titleAs || 'h3';
    var body = h('div', { className: 'bd-mcard-body' },
      p.tag ? h('div', { className: 'bd-mcard-tagrow' }, p.tag) : null,
      p.eyebrow ? h('div', { className: 'bd-mcard-eyebrow' }, p.eyebrow) : null,
      h(Title, { className: 'bd-mcard-title' }, p.href ? h('a', { href: p.href, className: 'bd-mcard-link' }, p.title) : p.title),
      p.description ? h('p', { className: 'bd-mcard-desc' }, p.description) : null,
      progress, author,
      p.actions ? h('div', { className: 'bd-mcard-actions' }, p.actions) : null);
    var classes = cx('bd-card', 'bd-glass', 'bd-card-tint-' + (p.tint || 'sheen'), 'bd-mcard', 'bd-mcard-' + layout, (p.interactive || p.onClick || p.href) && 'bd-card-interactive', p.animate && 'bd-enter', p.className);
    var style = Object.assign({ '--bd-ratio': ratio }, p.animate && p.delay ? { '--bd-delay': p.delay + 'ms' } : {}, p.style || {});
    return h('article', { className: classes, style: style, onClick: p.onClick, tabIndex: p.onClick ? 0 : undefined },
      h('div', { className: 'bd-mcard-media' }, media,
        p.badge != null ? h('span', { className: 'bd-mcard-badge' }, p.badge) : null,
        p.mediaAction ? h('span', { className: 'bd-mcard-mact' }, p.mediaAction) : null),
      body);
  }

  function SegmentedControl(p) {
    var opts = (p.options || []).map(function (o) { return typeof o === 'string' ? { value: o, label: o } : o; });
    var st = R.useState(p.defaultValue != null ? p.defaultValue : (opts[0] && opts[0].value));
    var value = p.value != null ? p.value : st[0];
    var ind = useIndicator(value, [opts.length, p.size]);
    function pick(v) { if (p.value == null) st[1](v); if (p.onChange) p.onChange(v); }
    function onKey(e) {
      var i = opts.findIndex(function (o) { return o.value === value; }), n = opts.length, j = -1;
      if (e.key === 'ArrowRight' || e.key === 'ArrowDown') j = (i + 1) % n;
      if (e.key === 'ArrowLeft' || e.key === 'ArrowUp') j = (i - 1 + n) % n;
      if (j < 0) return; e.preventDefault(); pick(opts[j].value);
      var btns = e.currentTarget.querySelectorAll('.bd-seg-opt'); if (btns[j]) btns[j].focus();
    }
    return h('div', { ref: ind.ref, role: 'radiogroup', 'aria-label': p.label, onKeyDown: onKey, className: cx('bd-seg', 'bd-glass', p.tone === 'accent' && 'bd-seg-accent', p.size === 'sm' && 'bd-seg-sm', p.className) },
      h('span', { className: cx('bd-seg-ind', ind.animated && 'bd-ind-anim'), style: ind.style, 'aria-hidden': 'true' }),
      opts.map(function (o) {
        var on = o.value === value;
        return h('button', { key: o.value, type: 'button', role: 'radio', 'aria-checked': String(on), 'data-bd-on': String(on), tabIndex: on ? 0 : -1, className: 'bd-seg-opt', onClick: function () { pick(o.value); } }, o.label);
      }));
  }

  function SearchField(p) {
    var rest = Object.assign({}, p); ['className', 'label'].forEach(function (k) { delete rest[k]; });
    return h('label', { className: cx('bd-search', 'bd-glass', p.className) },
      h('input', Object.assign({ type: 'search', placeholder: 'Search', 'aria-label': p.label || p.placeholder || 'Search' }, rest)),
      h(Icon, { name: 'search' }));
  }

  function Card(p) {
    var Tag = p.as || 'div';
    return h(Tag, { className: cx('bd-card', 'bd-glass', 'bd-card-tint-' + (p.tint || 'sheen'), p.compact && 'bd-card-compact', (p.interactive || p.onClick) && 'bd-card-interactive', p.elevation && 'bd-elev-' + p.elevation, p.animate && 'bd-enter', p.className), style: p.animate && p.delay ? Object.assign({ '--bd-delay': p.delay + 'ms' }, p.style) : p.style, onClick: p.onClick, tabIndex: p.onClick ? 0 : undefined },
      p.title ? h('h3', { className: 'bd-card-title' }, p.title) : null,
      p.description ? h('p', { className: 'bd-card-body' }, p.description) : null,
      p.children);
  }

  function initials(n) { return (n || '?').split(/\s+/).map(function (w) { return w[0]; }).slice(0, 2).join('').toUpperCase(); }
  function NotificationItem(p) {
    var av = p.avatar ? h('img', { className: 'bd-avatar', src: p.avatar, alt: '' }) : h('span', { className: 'bd-avatar', 'aria-hidden': 'true' }, initials(p.name));
    return h('div', { className: cx('bd-notif', 'bd-glass', p.animate && 'bd-enter', p.className), style: p.animate && p.delay ? { '--bd-delay': p.delay + 'ms' } : null },
      av,
      h('div', { className: 'bd-notif-text' }, h('div', { className: 'bd-notif-name' }, p.name), h('div', { className: 'bd-notif-action' }, p.action)),
      h('div', { className: 'bd-notif-time' }, p.time),
      p.unread ? h('span', { className: 'bd-unread', 'aria-label': 'Unread' }) : null);
  }

  function idOf(it) { return it.value != null ? it.value : it.label; }
  function containsId(it, v) { return (it.children || []).some(function (c) { return idOf(c) === v || containsId(c, v); }); }
  function MenuList(p) {
    var items = p.items || [];
    var st = R.useState(p.defaultValue != null ? p.defaultValue : null);
    var value = p.value != null ? p.value : st[0];
    var init = {};
    (function walk(list) { list.forEach(function (it) { if (it && it.children) { if (it.defaultOpen || containsId(it, value)) init[idOf(it)] = true; walk(it.children); } }); })(items);
    var fly = p.submenu !== 'inline';
    var fs = R.useState(null), flyOpen = fs[0], setFly = fs[1], timer = R.useRef(null);
    R.useEffect(function () { return function () { clearTimeout(timer.current); }; }, []);
    var os = R.useState(fly ? {} : init), open = os[0];
    function toggle(id) { var n = Object.assign({}, open); n[id] = !n[id]; os[1](n); }
    var ind = useIndicator(value, [items.length, JSON.stringify(open)]);
    function select(it) { var id = idOf(it); if (p.value == null) st[1](id); if (p.onSelect) p.onSelect(id, it); if (it.onSelect) it.onSelect(id); }
    function inner(it, extra) {
      return [it.icon ? h('span', { key: 'i', className: 'bd-menu-icon', 'aria-hidden': 'true' }, typeof it.icon === 'string' ? h(Icon, { name: it.icon }) : it.icon) : null,
        h('span', { key: 'l', className: 'bd-menu-label' }, it.label),
        it.badge != null ? h('span', { key: 'b', className: 'bd-menu-badge' }, it.badge) : null,
        extra || it.trailing || null];
    }
    function renderList(list, depth, inFly) {
      return list.map(function (it, i) {
        if (it === 'separator') return h('li', { key: 'sep' + i, className: 'bd-menu-sep', role: 'separator' });
        if (typeof it === 'string') return h('li', { key: 'h' + i, className: 'bd-menu-heading', role: 'presentation' }, it);
        var id = idOf(it), on = id === value;
        var cls = cx('bd-menu-item', depth > 0 && 'bd-menu-sub-item');
        if (it.children) {
          var childOn = containsId(it, value);
          var subId = 'bd-sub-' + String(id).replace(/[^A-Za-z0-9_-]/g, '');
          if (fly) {
            var fOpen = flyOpen === id;
            return h('li', { key: id, className: cx('bd-menu-group', 'bd-menu-fly-group', fOpen && 'bd-menu-fly-open'),
                onMouseEnter: function () { clearTimeout(timer.current); setFly(id); },
                onMouseLeave: function () { clearTimeout(timer.current); timer.current = setTimeout(function () { setFly(null); }, 160); },
                onKeyDown: function (e) {
                  if (e.key === 'Escape' || (e.key === 'ArrowLeft' && fOpen)) { e.preventDefault(); e.stopPropagation(); setFly(null); var b = e.currentTarget.querySelector('.bd-menu-item'); if (b) b.focus(); }
                  if (e.key === 'ArrowRight' && !fOpen) { e.preventDefault(); setFly(id); var li = e.currentTarget; setTimeout(function () { var f = li.querySelector('.bd-menu-fly .bd-menu-item'); if (f) f.focus(); }, 30); }
                  if ((e.key === 'ArrowDown' || e.key === 'ArrowUp') && fOpen && e.target.closest('.bd-menu-fly')) {
                    e.preventDefault(); var els = Array.prototype.slice.call(e.currentTarget.querySelectorAll('.bd-menu-fly .bd-menu-item')), k = els.indexOf(document.activeElement);
                    els[(k + (e.key === 'ArrowDown' ? 1 : -1) + els.length) % els.length].focus();
                  }
                },
                onBlur: function (e) { var li = e.currentTarget; setTimeout(function () { if (!li.contains(document.activeElement)) setFly(function (cur) { return cur === id ? null : cur; }); }, 0); } },
              h('button', { type: 'button', className: cx(cls, childOn && 'bd-menu-has-current'), 'aria-haspopup': 'true', 'aria-expanded': String(fOpen), 'aria-controls': subId, 'data-bd-on': String(childOn), onClick: function () { setFly(fOpen ? null : id); } },
                inner(it, h('span', { className: 'bd-menu-caret bd-menu-caret-side', 'aria-hidden': 'true' }, h(Icon, { name: 'chevron', size: 16 })))),
              h('div', { className: cx('bd-menu-fly', 'bd-glass'), id: subId, role: 'group', 'aria-label': it.label, inert: fOpen ? undefined : '' },
                h('div', { className: 'bd-menu-fly-title' }, it.label),
                h('ul', { className: 'bd-menu-fly-list', role: 'list' }, renderList(it.children, depth + 1, true))));
          }
          var isOpen = !!open[id];
          return h('li', { key: id, className: cx('bd-menu-group', isOpen && 'bd-menu-open') },
            h('button', { type: 'button', className: cx(cls, childOn && 'bd-menu-has-current'), 'aria-expanded': String(isOpen), 'aria-controls': subId, 'data-bd-on': String(childOn && !isOpen), onClick: function () { toggle(id); } },
              inner(it, h('span', { className: 'bd-menu-caret', 'aria-hidden': 'true' }, h(Icon, { name: 'chevronDown', size: 16 })))),
            h('div', { className: 'bd-menu-sub-wrap', id: subId, inert: isOpen ? undefined : '' },
              h('ul', { className: 'bd-menu-sub', role: 'list' }, renderList(it.children, depth + 1))));
        }
        var props = { className: cx(cls, inFly && 'bd-menu-fly-item'), 'aria-current': on ? (it.href ? 'page' : 'true') : undefined, 'data-bd-on': String(on && !inFly),
          onClick: function (e) { if (inFly) setFly(null); if (it.href && it.href !== '#' && !p.onSelect && !it.onSelect && p.value == null) { st[1](id); return; } if (it.href) e.preventDefault(); select(it); } };
        return h('li', { key: id }, it.href ? h('a', Object.assign({ href: it.href, target: it.external ? '_blank' : undefined, rel: it.external ? 'noopener noreferrer' : undefined }, props), inner(it, it.external ? h(Icon, { name: 'external', size: 16, className: 'bd-menu-ext' }) : null))
          : h('button', Object.assign({ type: 'button' }, props), inner(it)));
      });
    }
    return h('ul', { ref: ind.ref, className: cx('bd-menu', p.glass !== false && 'bd-glass', flyOpen != null && 'bd-menu-flying', p.className), 'aria-label': p.label },
      h('li', { className: cx('bd-menu-ind', ind.animated && 'bd-ind-anim'), style: ind.style, 'aria-hidden': 'true', role: 'presentation' }, h('span', { key: String(value), className: 'bd-menu-ind-bar' })),
      renderList(items, 0));
  }

  function UserMenu(p) {
    var u = p.user || {};
    var os = R.useState(!!p.defaultOpen), open = os[0], setOpen = os[1];
    var root = R.useRef(null), menuRef = R.useRef(null), byUser = R.useRef(false);
    var id = useId(p.id), place = p.placement || 'bottom-end';
    R.useEffect(function () {
      if (!open) return;
      function onDoc(e) { if (root.current && !root.current.contains(e.target)) setOpen(false); }
      function onKey(e) { if (e.key === 'Escape') { setOpen(false); var b = root.current && root.current.querySelector('.bd-user-btn'); if (b) b.focus(); } }
      document.addEventListener('mousedown', onDoc); document.addEventListener('keydown', onKey);
      if (byUser.current) { var first = menuRef.current && menuRef.current.querySelector('[role="menuitem"]'); if (first) first.focus(); }
      return function () { document.removeEventListener('mousedown', onDoc); document.removeEventListener('keydown', onKey); };
    }, [open]);
    function onMenuKey(e) {
      var els = Array.prototype.slice.call(menuRef.current.querySelectorAll('[role="menuitem"]')), i = els.indexOf(document.activeElement);
      if (e.key === 'ArrowDown') { e.preventDefault(); els[(i + 1) % els.length].focus(); }
      if (e.key === 'ArrowUp') { e.preventDefault(); els[(i - 1 + els.length) % els.length].focus(); }
      if (e.key === 'Home') { e.preventDefault(); els[0].focus(); }
      if (e.key === 'End') { e.preventDefault(); els[els.length - 1].focus(); }
      if (e.key === 'Tab') setOpen(false);
    }
    var avatar = u.avatar ? h('img', { className: 'bd-avatar bd-user-av', src: u.avatar, alt: '' }) : h('span', { className: 'bd-avatar bd-user-av', 'aria-hidden': 'true' }, initials(u.name));
    var compact = p.variant === 'avatar';
    var items = p.items || [{ label: 'Account', icon: 'user' }, { label: 'Billing', icon: 'card' }, 'separator', { label: 'Sign out', icon: 'signOut', tone: 'danger' }];
    return h('div', { ref: root, className: cx('bd-user', 'bd-user-' + place, open && 'bd-user-open', p.className) },
      h('button', { type: 'button', className: cx('bd-user-btn', compact ? 'bd-user-compact' : 'bd-glass'), 'aria-haspopup': 'menu', 'aria-expanded': String(open), 'aria-controls': id, 'aria-label': compact ? (u.name + ', account menu') : undefined, onClick: function () { byUser.current = true; setOpen(!open); } },
        h('span', { className: 'bd-user-avwrap' }, avatar, u.status ? h('span', { className: cx('bd-user-status', 'bd-user-status-' + u.status), 'aria-hidden': 'true' }) : null),
        compact ? null : h('span', { className: 'bd-user-text' }, h('span', { className: 'bd-user-name' }, u.name), u.role || u.email ? h('span', { className: 'bd-user-role' }, u.role || u.email) : null),
        compact ? null : h(Icon, { name: 'chevronDown', size: 16, className: 'bd-user-caret' })),
      h('div', { ref: menuRef, id: id, role: 'menu', 'aria-label': 'Account', className: cx('bd-user-menu', 'bd-glass'), onKeyDown: onMenuKey, inert: open ? undefined : '' },
        h('div', { className: 'bd-user-head' }, avatar, h('span', { className: 'bd-user-text' }, h('span', { className: 'bd-user-name' }, u.name), u.email ? h('span', { className: 'bd-user-role' }, u.email) : null)),
        h('div', { className: 'bd-user-sep', role: 'separator' }),
        items.map(function (it, i) {
          if (it === 'separator') return h('div', { key: 's' + i, className: 'bd-user-sep', role: 'separator' });
          var Tag = it.href ? 'a' : 'button';
          return h(Tag, { key: it.label, role: 'menuitem', tabIndex: open ? 0 : -1, href: it.href, type: it.href ? undefined : 'button', className: cx('bd-user-item', it.tone === 'danger' && 'bd-user-danger'),
            style: { '--bd-i': i }, onClick: function () { setOpen(false); if (it.onSelect) it.onSelect(); if (p.onSelect) p.onSelect(it.label, it); } },
            it.icon ? h('span', { className: 'bd-menu-icon', 'aria-hidden': 'true' }, typeof it.icon === 'string' ? h(Icon, { name: it.icon }) : it.icon) : null,
            h('span', { className: 'bd-menu-label' }, it.label),
            it.shortcut ? h('kbd', { className: 'bd-tip-kbd' }, it.shortcut) : null);
        })));
  }

  function TopBar(p) {
    var items = p.items || [];
    var st = R.useState(p.defaultValue != null ? p.defaultValue : (items[0] && idOf(items[0])));
    var value = p.value != null ? p.value : st[0];
    var ind = useIndicator(value, [items.length]);
    var ds = R.useState(null), drop = ds[0], setDrop = ds[1], timer = R.useRef(null);
    var ms = R.useState(false), mobile = ms[0], setMobile = ms[1];
    var sc = R.useState(false), scrolled = sc[0], setScrolled = sc[1];
    R.useEffect(function () {
      if (!p.sticky) return;
      function on() { setScrolled(window.scrollY > 4); } on();
      window.addEventListener('scroll', on, { passive: true }); return function () { window.removeEventListener('scroll', on); };
    }, [p.sticky]);
    R.useEffect(function () {
      if (drop == null) return;
      function k(e) { if (e.key === 'Escape') setDrop(null); }
      document.addEventListener('keydown', k); return function () { document.removeEventListener('keydown', k); };
    }, [drop]);
    function select(it) { var id = idOf(it); if (p.value == null) st[1](id); setDrop(null); setMobile(false); if (p.onSelect) p.onSelect(id, it); }
    function navItem(it, inMobile) {
      var id = idOf(it), on = id === value || containsId(it, value);
      var cls = cx('bd-top-link', on && 'bd-top-on');
      var body = [it.icon ? h('span', { key: 'i', className: 'bd-top-icon', 'aria-hidden': 'true' }, typeof it.icon === 'string' ? h(Icon, { name: it.icon, size: 18 }) : it.icon) : null, h('span', { key: 'l' }, it.label)];
      if (it.children && !inMobile) {
        var open = drop === id, did = 'bd-top-drop-' + String(id).replace(/[^A-Za-z0-9_-]/g, '');
        return h('li', { key: id, className: cx('bd-top-item', 'bd-top-has-drop', open && 'bd-top-drop-open'),
            onMouseEnter: function () { clearTimeout(timer.current); setDrop(id); }, onMouseLeave: function () { timer.current = setTimeout(function () { setDrop(null); }, 160); },
            onBlur: function (e) { var li = e.currentTarget; setTimeout(function () { if (!li.contains(document.activeElement)) setDrop(function (c) { return c === id ? null : c; }); }, 0); } },
          h('button', { type: 'button', className: cls, 'data-bd-on': String(on), 'aria-haspopup': 'true', 'aria-expanded': String(open), 'aria-controls': did, onClick: function () { setDrop(open ? null : id); },
              onKeyDown: function (e) { if (e.key === 'ArrowDown') { e.preventDefault(); setDrop(id); var li = e.currentTarget.parentNode; setTimeout(function () { var f = li.querySelector('.bd-top-drop a, .bd-top-drop button'); if (f) f.focus(); }, 30); } } },
            body, h('span', { className: 'bd-top-caret', 'aria-hidden': 'true' }, h(Icon, { name: 'chevronDown', size: 14 }))),
          h('div', { className: cx('bd-top-drop'), id: did, inert: open ? undefined : '' },
            h('ul', { role: 'list' }, it.children.map(function (c) {
              var cid = idOf(c), con = cid === value;
              var Tag = c.href ? 'a' : 'button';
              return h('li', { key: cid }, h(Tag, { href: c.href, type: c.href ? undefined : 'button', className: cx('bd-top-drop-item', con && 'bd-top-drop-on'), 'aria-current': con ? 'page' : undefined,
                  onClick: function (e) { if (c.href === '#' || p.onSelect) e.preventDefault(); select(c); },
                  onKeyDown: function (e) { if (e.key === 'ArrowDown' || e.key === 'ArrowUp') { e.preventDefault(); var els = Array.prototype.slice.call(e.currentTarget.closest('ul').querySelectorAll('a, button')), k = els.indexOf(e.currentTarget); els[(k + (e.key === 'ArrowDown' ? 1 : -1) + els.length) % els.length].focus(); } } },
                c.icon ? h('span', { className: 'bd-top-drop-icon', 'aria-hidden': 'true' }, typeof c.icon === 'string' ? h(Icon, { name: c.icon, size: 20 }) : c.icon) : null,
                h('span', { className: 'bd-top-drop-text' }, h('span', { className: 'bd-top-drop-label' }, c.label), c.description ? h('span', { className: 'bd-top-drop-desc' }, c.description) : null)));
            }))));
      }
      var Tag = it.href ? 'a' : 'button';
      var list = inMobile && it.children ? [it].concat(it.children) : [it];
      return list.map(function (x, n) {
        var xid = idOf(x), xon = xid === value;
        return h('li', { key: xid + (n ? '-c' : ''), className: cx('bd-top-item', n && 'bd-top-m-sub') }, h(x.href ? 'a' : 'button', { href: x.href, type: x.href ? undefined : 'button', className: cx('bd-top-link', (n ? xon : on) && 'bd-top-on'), 'data-bd-on': inMobile ? undefined : String(on), 'aria-current': xon ? 'page' : undefined,
          onClick: function (e) { if (x.children && inMobile) return; if (x.href === '#' || p.onSelect) e.preventDefault(); select(x); } }, n ? x.label : body));
      });
    }
    var brand = p.brand || {};
    return h('header', { className: cx('bd-top', 'bd-top-' + (p.variant || 'bar'), p.sticky && 'bd-top-sticky', scrolled && 'bd-top-scrolled', mobile && 'bd-top-mobile-open', p.className) },
      h('div', { className: cx('bd-top-inner', 'bd-glass') },
        h(brand.href ? 'a' : 'span', { className: 'bd-top-brand', href: brand.href }, brand.logo || h('span', { className: 'bd-top-mark', 'aria-hidden': 'true' }), h('span', null, brand.name || 'Brand')),
        h('nav', { className: 'bd-top-nav', 'aria-label': p.label || 'Main' },
          h('ul', { ref: ind.ref, className: 'bd-top-list' },
            h('li', { className: cx('bd-top-ind', ind.animated && 'bd-ind-anim'), style: ind.style, 'aria-hidden': 'true', role: 'presentation' }),
            items.map(function (it) { return navItem(it, false); }))),
        h('div', { className: 'bd-top-end' },
          p.search ? h('div', { className: 'bd-top-search' }, h(SearchField, Object.assign({ placeholder: 'Search' }, p.search === true ? {} : p.search))) : null,
          p.actions || null,
          p.user ? h(UserMenu, Object.assign({ variant: 'avatar' }, p.user)) : null,
          p.cta || null,
          items.length ? h(IconButton, { className: 'bd-top-burger', icon: mobile ? 'close' : 'menu', label: mobile ? 'Close menu' : 'Open menu', tooltip: false, variant: 'tinted', 'aria-expanded': String(mobile), onClick: function () { setMobile(!mobile); } }) : null)),
      h('div', { className: cx('bd-top-sheet', 'bd-glass'), inert: mobile ? undefined : '' },
        h('ul', { role: 'list' }, items.map(function (it) { return navItem(it, true); }))));
  }

  function PageHeader(p) {
    var crumbs = p.breadcrumbs || [];
    return h('div', { className: cx('bd-ph', p.className) },
      crumbs.length ? h('nav', { 'aria-label': 'Breadcrumb', className: 'bd-ph-crumbs' }, h('ol', null, crumbs.map(function (c, i) {
        var last = i === crumbs.length - 1;
        return h('li', { key: i }, last ? h('span', { 'aria-current': 'page' }, c.label) : h('a', { href: c.href || '#' }, c.label), last ? null : h(Icon, { name: 'chevron', size: 14, className: 'bd-ph-sep' }));
      }))) : null,
      h('div', { className: 'bd-ph-row' },
        h('div', { className: 'bd-ph-main' },
          p.eyebrow ? h('div', { className: 'bd-ph-eyebrow' }, p.eyebrow) : null,
          h(p.as || 'h1', { className: cx('bd-ph-title', p.gradient && ('bd-text-gradient' + (p.gradient === 'primary' ? '' : '-' + p.gradient))) }, p.title),
          p.subtitle ? h('p', { className: 'bd-ph-sub' }, p.subtitle) : null,
          p.meta ? h('div', { className: 'bd-ph-meta' }, p.meta) : null),
        p.actions ? h('div', { className: 'bd-ph-actions' }, p.actions) : null),
      p.tabs ? h('div', { className: 'bd-ph-tabs' }, p.tabs) : null);
  }

  function Toggle(p) {
    var st = R.useState(!!p.defaultChecked);
    var on = p.checked != null ? p.checked : st[0];
    return h('label', { className: cx('bd-toggle', p.className) },
      p.label ? h('span', null, p.label) : null,
      h('input', { type: 'checkbox', role: 'switch', className: 'bd-toggle-input', checked: on, disabled: p.disabled, onChange: function (e) { if (p.checked == null) st[1](e.target.checked); if (p.onChange) p.onChange(e.target.checked); } }),
      h('span', { className: 'bd-toggle-track', 'aria-hidden': 'true' }, h('span', { className: 'bd-toggle-thumb' })));
  }

  var uid = 0;
  function useId(given) { var r = R.useRef(null); if (r.current == null) r.current = given || ('bd-f' + (++uid)); return r.current; }
  function omit(p, keys) { var o = Object.assign({}, p); keys.forEach(function (k) { delete o[k]; }); return o; }
  function Field(id, p, control) {
    var msgId = id + '-msg';
    return h('div', { className: cx('bd-field', p.error && 'bd-field-invalid', p.disabled && 'bd-field-disabled', p.className) },
      p.label ? h('label', { className: 'bd-field-label', htmlFor: id }, p.label) : null,
      control(p.error || p.hint ? msgId : undefined),
      p.error ? h('div', { className: 'bd-field-msg bd-field-error', id: msgId, role: 'alert' }, p.error)
        : p.hint ? h('div', { className: 'bd-field-msg', id: msgId }, p.hint) : null);
  }

  function TextField(p) {
    var id = useId(p.id);
    var rest = omit(p, ['label', 'hint', 'error', 'icon', 'trailing', 'className', 'id']);
    return Field(id, p, function (desc) {
      return h('div', { className: cx('bd-input', 'bd-glass') },
        p.icon ? h('span', { className: 'bd-input-icon', 'aria-hidden': 'true' }, typeof p.icon === 'string' ? h(Icon, { name: p.icon }) : p.icon) : null,
        h('input', Object.assign({ type: 'text' }, rest, { id: id, 'aria-invalid': p.error ? 'true' : undefined, 'aria-describedby': desc })),
        p.trailing || null);
    });
  }

  function TextArea(p) {
    var id = useId(p.id);
    var rest = omit(p, ['label', 'hint', 'error', 'className', 'id']);
    return Field(id, p, function (desc) {
      return h('textarea', Object.assign({ rows: 4 }, rest, { id: id, className: cx('bd-textarea', 'bd-glass'), 'aria-invalid': p.error ? 'true' : undefined, 'aria-describedby': desc }));
    });
  }

  function Select(p) {
    var id = useId(p.id);
    var opts = (p.options || []).map(function (o) { return typeof o === 'string' ? { value: o, label: o } : o; });
    var rest = omit(p, ['label', 'hint', 'error', 'options', 'prefix', 'placeholder', 'className', 'id']);
    return Field(id, p, function (desc) {
      return h('div', { className: cx('bd-select', 'bd-glass') },
        p.prefix ? h('span', { className: 'bd-select-prefix', 'aria-hidden': 'true' }, p.prefix) : null,
        h('select', Object.assign({}, rest, { id: id, 'aria-invalid': p.error ? 'true' : undefined, 'aria-describedby': desc }),
          p.placeholder ? h('option', { value: '', disabled: true }, p.placeholder) : null,
          opts.map(function (o) { return h('option', { key: o.value, value: o.value }, o.label); })),
        h(Icon, { name: 'chevronDown', size: 20, className: 'bd-select-chev' }));
    });
  }

  function Checkbox(p) {
    var rest = omit(p, ['label', 'description', 'className', 'children']);
    return h('label', { className: cx('bd-check', p.disabled && 'bd-check-disabled', p.className) },
      h('input', Object.assign({ type: 'checkbox' }, rest, { className: 'bd-check-input' })),
      h('span', { className: 'bd-check-box', 'aria-hidden': 'true' }, h(Icon, { name: 'check', size: 14 })),
      h('span', { className: 'bd-check-text' }, h('span', null, p.label || p.children), p.description ? h('span', { className: 'bd-check-desc' }, p.description) : null));
  }

  function RadioGroup(p) {
    var name = useId(p.name);
    var opts = (p.options || []).map(function (o) { return typeof o === 'string' ? { value: o, label: o } : o; });
    var st = R.useState(p.defaultValue != null ? p.defaultValue : (opts[0] && opts[0].value));
    var value = p.value != null ? p.value : st[0];
    return h('div', { role: 'radiogroup', 'aria-label': p.label, className: cx('bd-radios', p.direction === 'row' && 'bd-radios-row', p.className) },
      opts.map(function (o) {
        return h('label', { key: o.value, className: cx('bd-check', 'bd-radio', (p.disabled || o.disabled) && 'bd-check-disabled') },
          h('input', { type: 'radio', name: name, value: o.value, checked: o.value === value, disabled: p.disabled || o.disabled, className: 'bd-check-input', onChange: function () { if (p.value == null) st[1](o.value); if (p.onChange) p.onChange(o.value); } }),
          h('span', { className: 'bd-check-box', 'aria-hidden': 'true' }, h('span', { className: 'bd-radio-dot' })),
          h('span', { className: 'bd-check-text' }, o.label));
      }));
  }

  function Slider(p) {
    var min = p.min != null ? p.min : 0, max = p.max != null ? p.max : 100, step = p.step || 1;
    var st = R.useState(p.defaultValue != null ? p.defaultValue : min);
    var value = p.value != null ? p.value : st[0];
    var pct = max > min ? (value - min) / (max - min) * 100 : 0;
    var fmt = p.format || function (v) { return String(v); };
    var id = useId(p.id);
    return h('div', { className: cx('bd-slider', p.disabled && 'bd-field-disabled', p.className) },
      (p.label || p.onClear) ? h('div', { className: 'bd-slider-head' },
        p.label ? h('label', { htmlFor: id }, p.label) : h('span'),
        p.onClear ? h('button', { type: 'button', className: 'bd-link', onClick: p.onClear }, 'Clear') : null) : null,
      h('div', { className: 'bd-slider-body', style: { '--bd-pct': pct + '%' } },
        p.showValue !== false ? h('span', { className: 'bd-slider-bubble', 'aria-hidden': 'true' }, fmt(value)) : null,
        h('input', { id: id, type: 'range', min: min, max: max, step: step, value: value, disabled: p.disabled, className: 'bd-range', 'aria-valuetext': fmt(value),
          onChange: function (e) { var v = Number(e.target.value); if (p.value == null) st[1](v); if (p.onChange) p.onChange(v); } })));
  }

  function Stepper(p) {
    var min = p.min != null ? p.min : 0, max = p.max != null ? p.max : 99;
    var st = R.useState(p.defaultValue != null ? p.defaultValue : min);
    var value = p.value != null ? p.value : st[0];
    function set(v) { v = Math.max(min, Math.min(max, v)); if (p.value == null) st[1](v); if (p.onChange) p.onChange(v); }
    var lbl = p.label || 'Quantity';
    return h('div', { className: cx('bd-stepper', p.className), role: 'group', 'aria-label': lbl },
      h('button', { type: 'button', className: 'bd-step bd-step-plus', 'aria-label': 'Increase ' + lbl, disabled: p.disabled || value >= max, onClick: function () { set(value + 1); } }, h(Icon, { name: 'plus', size: 16 })),
      h('output', { key: value, className: 'bd-step-value', 'aria-live': 'polite' }, value),
      h('button', { type: 'button', className: 'bd-step bd-step-minus', 'aria-label': 'Decrease ' + lbl, disabled: p.disabled || value <= min, onClick: function () { set(value - 1); } }, h(Icon, { name: 'minus', size: 16 })));
  }

  function FieldTile(p) {
    var rest = omit(p, ['label', 'value', 'icon', 'placeholder', 'className']);
    return h('button', Object.assign({ type: 'button' }, rest, { className: cx('bd-tile', 'bd-glass', p.className) }),
      p.icon ? h('span', { className: 'bd-tile-icon', 'aria-hidden': 'true' }, typeof p.icon === 'string' ? h(Icon, { name: p.icon, size: 24 }) : p.icon) : null,
      h('span', { className: 'bd-tile-text' },
        h('span', { className: 'bd-tile-label' }, p.label),
        h('span', { className: cx('bd-tile-value', !p.value && 'bd-tile-empty') }, p.value || p.placeholder)));
  }

  function Tooltip(p) {
    var id = useId(p.id);
    var st = R.useState(false), open = st[0], set = st[1];
    var place = p.placement || 'top';
    var child = R.Children.only(p.children);
    var trigger = R.cloneElement(child, { 'aria-describedby': open ? id : undefined });
    return h('span', { className: 'bd-tip-wrap', onMouseEnter: function () { set(true); }, onMouseLeave: function () { set(false); }, onFocus: function () { set(true); }, onBlur: function () { set(false); }, onKeyDown: function (e) { if (e.key === 'Escape') set(false); } },
      trigger,
      h('span', { role: 'tooltip', id: id, className: cx('bd-tip', 'bd-tip-' + place, (open || p.open) && 'bd-tip-open') },
        p.content, p.shortcut ? h('kbd', { className: 'bd-tip-kbd' }, p.shortcut) : null));
  }

  function IconButton(p) {
    var variant = p.variant || 'glass', size = p.size || 'md', shape = p.shape || 'circle';
    var rest = omit(p, ['variant', 'size', 'shape', 'icon', 'label', 'tooltip', 'glow', 'pressed', 'badge', 'className']);
    var px = { sm: 16, md: 20, lg: 24, xl: 32 }[size] || 20;
    var btn = h('button', Object.assign({ type: 'button' }, rest, {
      'aria-label': p.label, 'aria-pressed': p.pressed != null ? String(!!p.pressed) : undefined,
      className: cx('bd-ibtn', 'bd-btn-' + variant, 'bd-ibtn-' + size, 'bd-ibtn-' + shape, p.glow && 'bd-btn-glow', p.className)
    }), iconNode(p.icon, px), p.badge != null ? h('span', { className: 'bd-ibtn-badge', 'aria-hidden': 'true' }, p.badge === true ? '' : p.badge) : null);
    return p.tooltip === false ? btn : h(Tooltip, { content: p.tooltip || p.label, placement: p.tooltipPlacement, shortcut: p.shortcut }, btn);
  }

  function ButtonGroup(p) {
    return h('div', { role: 'group', 'aria-label': p.label, className: cx('bd-bgroup', 'bd-glass', p.size === 'sm' && 'bd-bgroup-sm', p.className) }, p.children);
  }

  function Tag(p) {
    var tone = p.tone || 'neutral', variant = p.variant || 'dot';
    return h('span', { className: cx('bd-tag', 'bd-tag-' + variant, 'bd-tone-' + tone, p.className) },
      variant === 'dot' ? h('span', { className: 'bd-tag-dot', 'aria-hidden': 'true' }) : null, p.children);
  }

  function cmp(a, b) {
    if (a == null) return 1; if (b == null) return -1;
    if (typeof a === 'number' && typeof b === 'number') return a - b;
    return String(a).localeCompare(String(b), undefined, { numeric: true, sensitivity: 'base' });
  }
  function MediaCell(p) {
    var shape = p.shape || 'circle', size = p.size || 'md';
    var img = p.image, media;
    if (img && /^gradient-/.test(img)) media = h('span', { className: 'bd-mc-img', style: { background: 'var(--' + img + ')' }, 'aria-hidden': 'true' });
    else if (img) media = h('img', { className: 'bd-mc-img', src: img, alt: p.imageAlt || '' });
    else media = h('span', { className: 'bd-mc-img bd-mc-initials', 'aria-hidden': 'true' }, initials(typeof p.title === 'string' ? p.title : ''));
    return h('span', { className: cx('bd-mc', 'bd-mc-' + shape, 'bd-mc-' + size, p.className) },
      h('span', { className: 'bd-mc-media' }, media, p.badge != null ? h('span', { className: 'bd-mc-badge' }, p.badge) : null),
      h('span', { className: 'bd-mc-text' },
        h('span', { className: 'bd-mc-title' }, p.title),
        p.subtitle ? h('span', { className: 'bd-mc-sub' }, p.subtitle) : null,
        p.meta ? h('span', { className: 'bd-mc-meta' }, p.meta) : null));
  }
  function pick(spec, row) { return spec == null ? undefined : typeof spec === 'function' ? spec(row) : row[spec]; }

  function DataTable(p) {
    var cols = p.columns || [], rows = p.rows || [];
    var keyOf = typeof p.rowKey === 'function' ? p.rowKey : function (r) { return r[p.rowKey || 'id']; };
    var ss = R.useState(p.defaultSort || null); var sort = p.sort !== undefined ? p.sort : ss[0];
    var sel = R.useState(p.defaultSelected || []); var selected = p.selected || sel[0];
    function setSort(k) {
      var next = !sort || sort.key !== k ? { key: k, dir: 'asc' } : sort.dir === 'asc' ? { key: k, dir: 'desc' } : null;
      if (p.sort === undefined) ss[1](next); if (p.onSortChange) p.onSortChange(next);
    }
    function setSel(next) { if (!p.selected) sel[1](next); if (p.onSelectionChange) p.onSelectionChange(next); }
    var view = rows;
    if (sort && !p.manualSort) {
      var col = cols.filter(function (c) { return c.key === sort.key; })[0] || {};
      var get = col.sortValue || function (r) { return r[sort.key]; };
      view = rows.slice().sort(function (a, b) { var d = cmp(get(a), get(b)); return sort.dir === 'desc' ? -d : d; });
    }
    var keys = view.map(keyOf);
    var nSel = keys.filter(function (k) { return selected.indexOf(k) >= 0; }).length;
    var all = keys.length > 0 && nSel === keys.length, some = nSel > 0 && !all;
    var allRef = R.useRef(null);
    R.useEffect(function () { if (allRef.current) allRef.current.indeterminate = some; });
    function toggle(k) { setSel(selected.indexOf(k) >= 0 ? selected.filter(function (x) { return x !== k; }) : selected.concat([k])); }
    function check(props) {
      return h('label', { className: 'bd-check bd-dt-check' },
        h('input', Object.assign({ type: 'checkbox', className: 'bd-check-input' }, props)),
        h('span', { className: 'bd-check-box bd-check-box-sq', 'aria-hidden': 'true' }, h(Icon, { name: props.ref ? (some ? 'minus' : 'check') : 'check', size: 14 })));
    }
    var head = h('tr', null,
      p.selectable ? h('th', { className: 'bd-dt-sel', scope: 'col' }, check({ ref: allRef, checked: all, 'aria-label': 'Select all rows', onChange: function () { setSel(all ? [] : keys); } })) : null,
      cols.map(function (c) {
        var active = sort && sort.key === c.key;
        var aria = active ? (sort.dir === 'asc' ? 'ascending' : 'descending') : (c.sortable ? 'none' : undefined);
        var label = c.sortable ? h('button', { type: 'button', className: cx('bd-dt-sort', active && 'bd-dt-sorted'), onClick: function () { setSort(c.key); } },
          c.header, h(Icon, { name: active ? (sort.dir === 'asc' ? 'sortUp' : 'sortDown') : 'sort', size: 16 })) : c.header;
        return h('th', { key: c.key, scope: 'col', 'aria-sort': aria, className: cx(c.align && 'bd-dt-' + c.align), style: c.width ? { width: c.width } : null }, label);
      }),
      p.rowActions ? h('th', { className: 'bd-dt-actions', scope: 'col' }, h('span', { className: 'bd-sr' }, 'Actions')) : null);
    var body = view.length ? view.map(function (r) {
      var k = keyOf(r), on = selected.indexOf(k) >= 0;
      return h('tr', { key: k, 'aria-selected': p.selectable ? String(on) : undefined, className: on ? 'bd-dt-row-on' : null },
        p.selectable ? h('td', { className: 'bd-dt-sel' }, check({ checked: on, 'aria-label': 'Select row ' + (p.rowLabel ? p.rowLabel(r) : k), onChange: function () { toggle(k); } })) : null,
        cols.map(function (c) {
          var v = c.render ? c.render(r) : r[c.key];
          if (c.media && !c.render) { var m = c.media; v = h(MediaCell, { title: v, image: pick(m.image, r), subtitle: pick(m.subtitle, r), meta: pick(m.meta, r), badge: pick(m.badge, r), shape: m.shape, size: m.size }); }
          else if (c.subtitle && !c.render) { v = h('span', { className: 'bd-dt-two' }, h('span', { className: 'bd-dt-two-main' }, v), h('span', { className: 'bd-dt-two-sub' }, pick(c.subtitle, r))); }
          if (c.tags && !c.render) { var tv = Array.isArray(v) ? v : [v]; v = h('span', { className: 'bd-dt-tags' }, tv.map(function (t, i) { var o = typeof c.tags === 'function' ? c.tags(t, r) : {}; return h(Tag, Object.assign({ key: i }, o), t); })); }
          return h('td', { key: c.key, className: cx(c.align && 'bd-dt-' + c.align, c.primary && 'bd-dt-primary', c.wrap && 'bd-dt-wrap'), style: c.maxWidth ? { maxWidth: c.maxWidth } : null }, v);
        }),
        p.rowActions ? h('td', { className: 'bd-dt-actions' }, h('span', { className: 'bd-dt-actions-in' }, p.rowActions(r))) : null);
    }) : [h('tr', { key: 'empty' }, h('td', { className: 'bd-dt-empty', colSpan: cols.length + (p.selectable ? 1 : 0) + (p.rowActions ? 1 : 0) }, p.empty || 'No results'))];
    var bar = (p.title || p.toolbar || (p.selectable && nSel > 0)) ? h('div', { className: cx('bd-dt-bar', nSel > 0 && 'bd-dt-bar-sel') },
      nSel > 0 && p.selectable
        ? [h('span', { key: 'n', className: 'bd-dt-count', 'aria-live': 'polite' }, nSel + ' selected'),
           h('span', { key: 'a', className: 'bd-dt-bulk' }, p.bulkActions ? p.bulkActions(keys.filter(function (x) { return selected.indexOf(x) >= 0; }), function () { setSel([]); }) : null,
             h(Button, { variant: 'tinted', size: 'sm', onClick: function () { setSel([]); } }, 'Clear'))]
        : [h('span', { key: 't', className: 'bd-dt-title' }, p.title), h('span', { key: 'tb', className: 'bd-dt-bulk' }, p.toolbar || null)]) : null;
    return h('div', { className: cx('bd-dt', 'bd-glass', p.density === 'compact' && 'bd-dt-compact', p.density === 'spacious' && 'bd-dt-spacious', p.className) },
      bar,
      h('div', { className: 'bd-dt-scroll' }, h('table', null, p.caption ? h('caption', { className: 'bd-sr' }, p.caption) : null, h('thead', null, head), h('tbody', null, body))),
      p.footer ? h('div', { className: 'bd-dt-foot' }, p.footer) : null);
  }

  function pageList(page, count, sib) {
    var out = [], lo = Math.max(2, page - sib), hi = Math.min(count - 1, page + sib);
    if (page - sib <= 3) { lo = 2; hi = Math.min(count - 1, Math.max(hi, 2 + sib * 2)); }
    if (page + sib >= count - 2) { hi = count - 1; lo = Math.max(2, Math.min(lo, count - 1 - sib * 2)); }
    out.push(1); if (lo > 2) out.push('gap-l');
    for (var i = lo; i <= hi; i++) out.push(i);
    if (hi < count - 1) out.push('gap-r'); if (count > 1) out.push(count);
    return out;
  }
  function Pagination(p) {
    var count = Math.max(1, p.pageCount || 1);
    var st = R.useState(p.defaultPage || 1); var page = p.page || st[0];
    function go(n) { n = Math.max(1, Math.min(count, n)); if (!p.page) st[1](n); if (p.onChange) p.onChange(n); }
    var sib = p.siblingCount != null ? p.siblingCount : 1;
    var nav = p.variant === 'simple'
      ? h('div', { className: 'bd-pg-simple' },
          h(IconButton, { icon: 'chevron', label: 'Previous page', className: 'bd-pg-prev', disabled: page <= 1, onClick: function () { go(page - 1); }, size: 'sm' }),
          h('span', { className: 'bd-pg-of', 'aria-live': 'polite' }, page + ' of ' + count),
          h(IconButton, { icon: 'chevron', label: 'Next page', disabled: page >= count, onClick: function () { go(page + 1); }, size: 'sm' }))
      : h('div', { className: cx('bd-pg-pages', 'bd-glass') },
          h('button', { type: 'button', className: 'bd-pg-btn bd-pg-arrow', 'aria-label': 'Previous page', disabled: page <= 1, onClick: function () { go(page - 1); } }, h(Icon, { name: 'arrowLeft', size: 16 })),
          pageList(page, count, sib).map(function (n) {
            return typeof n === 'string' ? h('span', { key: n, className: 'bd-pg-gap', 'aria-hidden': 'true' }, '…')
              : h('button', { key: n, type: 'button', className: 'bd-pg-btn', 'aria-current': n === page ? 'page' : undefined, 'aria-label': 'Page ' + n, onClick: function () { go(n); } }, n);
          }),
          h('button', { type: 'button', className: 'bd-pg-btn bd-pg-arrow', 'aria-label': 'Next page', disabled: page >= count, onClick: function () { go(page + 1); } }, h(Icon, { name: 'arrowRight', size: 16 })));
    var summary = null;
    if (p.total != null && p.pageSize) { var a = (page - 1) * p.pageSize + 1, b = Math.min(p.total, page * p.pageSize); summary = h('span', { className: 'bd-pg-sum' }, (p.total ? a : 0) + '–' + b + ' of ' + p.total); }
    var size = p.pageSizeOptions ? h('label', { className: 'bd-pg-size' }, 'Rows per page',
      h(Select, { 'aria-label': 'Rows per page', options: p.pageSizeOptions.map(String), value: String(p.pageSize), onChange: function (e) { if (p.onPageSizeChange) p.onPageSizeChange(Number(e.target.value)); } })) : null;
    return h('nav', { 'aria-label': p.label || 'Pagination', className: cx('bd-pg', p.className) }, summary || size ? h('div', { className: 'bd-pg-meta' }, size, summary) : null, nav);
  }

  window.BragDocUI = Object.assign(window.BragDocUI || {}, { Button: Button, IconButton: IconButton, ButtonGroup: ButtonGroup, Tooltip: Tooltip, SegmentedControl: SegmentedControl, SearchField: SearchField, Card: Card, MediaCard: MediaCard, HeroHeader: HeroHeader, NotificationItem: NotificationItem, MenuList: MenuList, UserMenu: UserMenu, TopBar: TopBar, PageHeader: PageHeader, Toggle: Toggle, TextField: TextField, TextArea: TextArea, Select: Select, Checkbox: Checkbox, RadioGroup: RadioGroup, Slider: Slider, Stepper: Stepper, FieldTile: FieldTile, Tag: Tag, MediaCell: MediaCell, DataTable: DataTable, Pagination: Pagination, Icon: Icon });
})();
