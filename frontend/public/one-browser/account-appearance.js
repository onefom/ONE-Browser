(function(root){
  'use strict';
  function normalize(accounts){
    const used=new Set();
    for(const account of accounts){
      if(typeof account.colorHue==='number'&&Number.isFinite(account.colorHue)&&account.colorHue>=0&&account.colorHue<360&&!used.has(account.colorHue))used.add(account.colorHue);
      else delete account.colorHue;
      delete account.tone;
    }
    let seed=0;
    for(const account of accounts){
      if(typeof account.colorHue==='number')continue;
      let hue;
      do{hue=Math.round(((227+seed++*137.508)%360)*1000)/1000}while(used.has(hue));
      account.colorHue=hue;used.add(hue);
    }
    return accounts;
  }
  function badgeLabel(value){const text=String(value||'').trim();if(/^\d+$/.test(text))return text.length<=4?text:text.slice(0,3)+'…';return Array.from(text)[0]?.toUpperCase()||'?'}
  root.OneBrowserAccountAppearance={normalize,badgeLabel};
})(typeof window!=='undefined'?window:globalThis);
