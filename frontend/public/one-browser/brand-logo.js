(function(root){
  'use strict';
  const MAX_FILE_BYTES=5*1024*1024;
  const FORMATS=new Set(['image/jpeg','image/png','image/webp','image/gif','image/bmp','image/x-ms-bmp','image/avif']);
  const validLogo=value=>typeof value==='string'&&value.length<=2*1024*1024&&/^data:image\/png;base64,[A-Za-z0-9+/=]+$/.test(value);
  function cropGeometry(width,height,size,zoom,x,y){
    const scale=Math.max(size/width,size/height)*zoom,w=width*scale,h=height*scale;
    return {width:w,height:h,x:Math.max((size-w)/2,Math.min((w-size)/2,x)),y:Math.max((size-h)/2,Math.min((h-size)/2,y))};
  }
  function init(notify){
    const $=id=>document.getElementById(id),mark=$('brandMark'),preview=$('brandLogoPreview'),input=$('brandLogoFile'),modal=$('brandCropModal'),canvas=$('brandCropCanvas'),zoom=$('brandCropZoom');
    let saved={name:'Online',logo:''};
    try{const value=JSON.parse(localStorage.getItem('online-brand')||'null');if(value&&typeof value.name==='string')saved={name:value.name||'Online',logo:validLogo(value.logo)?value.logo:''}}catch(_){}
    let draftLogo=saved.logo,source=null,sourceURL='',generation=0,x=0,y=0,drag=null,returnFocus=null;
    function paintMark(element,logo){
      element.replaceChildren();element.classList.toggle('has-custom-logo',Boolean(logo));
      if(logo){const image=document.createElement('img');image.src=logo;image.alt='品牌标识';element.appendChild(image)}
    }
    function paintPreview(){paintMark(preview,draftLogo);$('resetBrandLogo').disabled=!draftLogo}
    function applyBrand(value){$('brandName').textContent=value.name.toUpperCase();paintMark(mark,value.logo)}
    applyBrand(saved);$('productName').value=saved.name;paintPreview();
    function close(){
      generation++;modal.classList.remove('open');modal.setAttribute('aria-hidden','true');source=null;drag=null;
      if(sourceURL){URL.revokeObjectURL(sourceURL);sourceURL=''}
      returnFocus?.focus();
    }
    function draw(){
      if(!source)return;
      const g=cropGeometry(source.naturalWidth,source.naturalHeight,canvas.width,Number(zoom.value),x,y);x=g.x;y=g.y;
      const ctx=canvas.getContext('2d');ctx.clearRect(0,0,canvas.width,canvas.height);
      ctx.drawImage(source,(canvas.width-g.width)/2+x,(canvas.height-g.height)/2+y,g.width,g.height);
      $('brandCropZoomValue').textContent=Math.round(Number(zoom.value)*100)+'%';
    }
    async function selectFile(file){
      if(!file)return;
      if(file.size>MAX_FILE_BYTES){notify('图片不能超过 5 MB');return}
      const type=(file.type||'').toLowerCase();
      if(type?!FORMATS.has(type):!(/\.(jpe?g|png|webp|gif|bmp|avif)$/i.test(file.name))){notify('请选择 JPG、PNG、WebP、GIF、BMP 或 AVIF 图片');return}
      if(sourceURL)URL.revokeObjectURL(sourceURL);
      const token=++generation,url=URL.createObjectURL(file),image=new Image();sourceURL=url;
      image.src=url;
      try{
        await image.decode();if(token!==generation)return;
        if(!image.naturalWidth||!image.naturalHeight||image.naturalWidth*image.naturalHeight>80000000)throw new Error('图片尺寸过大，请缩小后再试');
        source=image;x=y=0;zoom.value='1';draw();returnFocus=document.activeElement;
        modal.classList.add('open');modal.setAttribute('aria-hidden','false');$('closeBrandCrop').focus();
      }catch(error){if(token===generation){URL.revokeObjectURL(url);sourceURL='';notify(error.message==='图片尺寸过大，请缩小后再试'?error.message:'图片无法读取，请换一张图片重试')}}
    }
    $('chooseBrandLogo').onclick=()=>input.click();
    input.onchange=()=>{const file=input.files?.[0];input.value='';selectFile(file)};
    $('resetBrandLogo').onclick=()=>{draftLogo='';paintPreview();notify('已恢复默认标识，点击「保存品牌」生效')};
    $('saveBrand').onclick=()=>{
      const value={name:$('productName').value.trim()||'Online',logo:draftLogo};
      try{localStorage.setItem('online-brand',JSON.stringify(value));saved=value;applyBrand(value);$('productName').value=value.name;notify('品牌设置已保存')}
      catch(_){notify('品牌保存失败，存储空间不足，请清理后重试')}
    };
    zoom.oninput=draw;
    canvas.onpointerdown=event=>{if(!source||event.button!==0)return;event.preventDefault();const rect=canvas.getBoundingClientRect();drag={id:event.pointerId,x:event.clientX,y:event.clientY,offsetX:x,offsetY:y,ratio:canvas.width/rect.width};canvas.setPointerCapture(event.pointerId)};
    canvas.onpointermove=event=>{if(!drag||drag.id!==event.pointerId)return;x=drag.offsetX+(event.clientX-drag.x)*drag.ratio;y=drag.offsetY+(event.clientY-drag.y)*drag.ratio;draw()};
    canvas.onpointerup=canvas.onpointercancel=()=>{drag=null};
    canvas.onkeydown=event=>{const moves={ArrowLeft:[-8,0],ArrowRight:[8,0],ArrowUp:[0,-8],ArrowDown:[0,8]};if(!moves[event.key])return;event.preventDefault();x+=moves[event.key][0];y+=moves[event.key][1];draw()};
    $('resetBrandCrop').onclick=()=>{x=y=0;zoom.value='1';draw()};
    $('confirmBrandCrop').onclick=()=>{
      if(!source)return;
      const output=document.createElement('canvas');output.width=output.height=256;
      const ctx=output.getContext('2d');ctx.beginPath();ctx.arc(128,128,128,0,Math.PI*2);ctx.clip();ctx.drawImage(canvas,0,0,256,256);
      draftLogo=output.toDataURL('image/png');paintPreview();close();notify('裁切完成，点击「保存品牌」生效');
    };
    $('closeBrandCrop').onclick=$('cancelBrandCrop').onclick=close;
    modal.onclick=event=>{if(event.target===modal)close()};
    modal.addEventListener('keydown',event=>{
      if(event.key==='Escape'){event.preventDefault();event.stopImmediatePropagation();close();return}
      if(event.key!=='Tab')return;
      const focusable=[...modal.querySelectorAll('button,input,[tabindex="0"]')].filter(element=>!element.disabled);
      const first=focusable[0],last=focusable.at(-1);
      if(event.shiftKey&&document.activeElement===first){event.preventDefault();last.focus()}
      else if(!event.shiftKey&&document.activeElement===last){event.preventDefault();first.focus()}
    });
  }
  root.OneBrowserBrand={init,cropGeometry,validLogo,MAX_FILE_BYTES};
})(typeof window!=='undefined'?window:globalThis);
