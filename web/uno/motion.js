/* Card motion uses only the recipient's own hand and public counts/discards. */
window.UnoMotion=(()=>{
 const reduced=()=>matchMedia('(prefers-reduced-motion: reduce)').matches;
 const flights=new Set(),landing=new Map();let selected=null,suppressClickUntil=0;
 const rect=el=>el?.getBoundingClientRect();
 function capture(){return {cards:new Map([...document.querySelectorAll('[data-card]')].map(el=>[el.dataset.card,{rect:rect(el),html:el.outerHTML}])),players:new Map([...document.querySelectorAll('[data-player]')].map(el=>[el.dataset.player,rect(el)])),deck:rect(document.querySelector('.draw-pile')),scroll:document.querySelector('.hand')?.scrollLeft||0};}
 function angle(id){let n=0;for(const c of id)n=(n*31+c.charCodeAt(0))>>>0;return {x:(n%17)-8,y:((n>>>4)%13)-6,angle:((n>>>8)%23)-11};}
 function discardHTML(cards,large,draw){return `<div class="discard-pile ${large?'large':''}" aria-label="Pila de descarte">${cards.filter(Boolean).map((c,i)=>{const a=angle(c.id);return `<div class="discard-slot" data-discard="${c.id}" data-angle="${a.angle}" style="--pile-x:${a.x}px;--pile-y:${a.y}px;--pile-angle:${a.angle}deg;z-index:${i}">${draw(c,large)}</div>`}).join('')}</div>`;}
 function fly(from,to,html,{delay=0,flip=false,rotation=0,hide=null}={}){
  if(!from||!to||reduced()||!document.body.animate)return;
  const el=document.createElement('div');el.className='card-flight';el.setAttribute('aria-hidden','true');
  el.style.cssText=`left:${from.left}px;top:${from.top}px;width:${from.width}px;height:${from.height}px`;
  const inner=document.createElement('div');inner.className='flight-inner';
  const front=document.createElement('div');front.className='flight-face flight-front';front.innerHTML=html;
  front.querySelectorAll('button').forEach(b=>{b.disabled=false;b.removeAttribute('data-card');b.tabIndex=-1});
  const back=document.createElement('div');back.className='flight-face flight-back';back.innerHTML='<div class="card back"><span>UNO</span></div>';
  inner.append(front,back);el.append(inner);document.body.append(el);flights.add(el);
  if(hide){landing.set(hide,(landing.get(hide)||0)+1);hideTarget(hide,true);}
  const dx=to.left-from.left,dy=to.top-from.top,sx=to.width/from.width,sy=to.height/from.height;
  const animation=el.animate([{transform:'translate(0,0) rotate(-8deg) scale(1)',opacity:1},{transform:`translate(${dx*.48}px,${dy*.48-45}px) rotate(8deg) scale(${(1+sx)/2},${(1+sy)/2})`,offset:.48,opacity:1},{transform:`translate(${dx}px,${dy}px) rotate(${rotation}deg) scale(${sx},${sy})`,opacity:1}],{duration:650,delay,easing:'cubic-bezier(.22,.7,.25,1)',fill:'both'});
  if(flip)inner.animate([{transform:'rotateY(180deg)'},{transform:'rotateY(180deg)',offset:.4},{transform:'rotateY(0deg)'}],{duration:650,delay,fill:'both'});
  function cleanup(){el.remove();flights.delete(el);if(hide){const n=(landing.get(hide)||1)-1;if(n>0)landing.set(hide,n);else{landing.delete(hide);hideTarget(hide,false)}}}
  animation.finished.then(cleanup,cleanup);
 }
 function hideTarget(id,hide){document.querySelectorAll('[data-card],[data-discard]').forEach(el=>{if((el.dataset.card||el.dataset.discard)===id)el.classList.toggle('motion-landing',hide)});}
 function backHTML(){return '<div class="card back"><span>UNO</span></div>';}
 function afterUpdate(previous,next,before,draw){
  if(!previous||!before||next.phase==='lobby'||previous.revision===next.revision)return;
  const deck=rect(document.querySelector('.draw-pile'));if(!deck)return;
  const deal=previous.phase==='lobby'||previous.round!==next.round;
  const oldIDs=new Set(previous.hand.map(c=>c.id));
  const received=next.hand.filter(c=>!oldIDs.has(c.id));
  const hand=document.querySelector('.hand');if(received.length&&hand&&!deal)hand.scrollLeft=hand.scrollWidth;
  received.forEach((c,i)=>{const target=document.querySelector(`[data-card="${c.id}"]`);fly(deck,rect(target),draw(c,false),{delay:i*110,flip:true,hide:c.id});});
  const oldPlayers=new Map(previous.players.map(p=>[p.id,p]));
  for(const p of next.players){if(p.id===next.me)continue;const old=oldPlayers.get(p.id);const added=deal?p.count:Math.max(0,p.count-(old?.count||0));const target=rect(document.querySelector(`[data-player="${p.id}"]`));if(!target)continue;for(let i=0;i<Math.min(added,deal?7:4);i++){const to={left:target.left+target.width/2-24,top:target.top+12,width:48,height:72};fly(deck,to,backHTML(),{delay:i*90});}}
  if(next.top?.id!==previous.top?.id){const slot=document.querySelector(`[data-discard="${next.top.id}"]`);const target=slot?.querySelector('.card');const actor=previous.turn;const from=before.cards.get(next.top.id)?.rect||before.players.get(actor)||deck;fly(deal?deck:from,rect(target),draw(next.top,!!target?.classList.contains('large')),{delay:deal?500:0,rotation:Number(slot?.dataset.angle||0),hide:next.top.id});}
 }
 function select(el){if(selected===el)return;selected?.classList.remove('inspecting');selected=el;selected?.classList.add('inspecting');const label=document.getElementById('hand-selection'),button=document.getElementById('inspect-play');if(label)label.textContent=selected?.getAttribute('aria-label')||'Desliza el dedo sobre las cartas para verlas.';if(button)button.disabled=!selected||selected.dataset.playable!=='true';}
 function bindHand(play){
  const hand=document.querySelector('.hand');selected=null;if(!hand)return;
  hand.style.setProperty('--hand-count',hand.children.length);document.getElementById('inspect-play')?.addEventListener('click',()=>{if(selected?.dataset.playable==='true')play(selected.dataset.card)});
  [...hand.children].forEach((el,i)=>{el.style.setProperty('--card-index',i);if(landing.has(el.dataset.card))el.classList.add('motion-landing')});
  let gesture=null,lastTouchAt=-Infinity;
  hand.addEventListener('pointerdown',e=>{if(e.pointerType==='mouse')return;lastTouchAt=performance.now();const el=e.target.closest('[data-card]');if(!el)return;gesture={x:e.clientX,y:e.clientY,moved:false};hand.setPointerCapture(e.pointerId);select(el);});
  hand.addEventListener('pointermove',e=>{
   if(e.pointerType==='mouse'){if(performance.now()-lastTouchAt>=800)select(e.target.closest('[data-card]'));return;}
   if(!gesture)return;
   if(Math.abs(e.clientX-gesture.x)>9)gesture.moved=true;
   if(Math.abs(e.clientY-gesture.y)>28&&!gesture.moved){select(null);return;}
   const cards=[...hand.querySelectorAll('[data-card]')];
   let closest=null,dist=Infinity;for(const card of cards){const r=card.getBoundingClientRect(),d=Math.abs(e.clientX-(r.left+Math.min(r.width,35)/2));if(d<dist){dist=d;closest=card}}
   select(closest);
   const r=hand.getBoundingClientRect();if(e.clientX>r.right-25)hand.scrollLeft+=12;if(e.clientX<r.left+25)hand.scrollLeft-=12;
  });
  hand.addEventListener('pointerup',e=>{if(!gesture)return;suppressClickUntil=performance.now()+400;const chosen=selected;const moved=gesture.moved;gesture=null;try{hand.releasePointerCapture(e.pointerId)}catch{}if(!moved&&chosen)chosen.focus({preventScroll:true});});
  hand.addEventListener('pointercancel',()=>{gesture=null;select(null)});
  hand.addEventListener('pointerleave',e=>{if(e.pointerType==='mouse'&&performance.now()-lastTouchAt>=800)select(null)});
  hand.addEventListener('click',e=>{if(performance.now()<suppressClickUntil){e.preventDefault();e.stopImmediatePropagation();}},{capture:true});
  hand.addEventListener('focusin',e=>{if(!gesture)select(e.target.closest('[data-card]'))});
 }
 return {capture,discardHTML,afterUpdate,bindHand,get busy(){return flights.size>0}};
})();
