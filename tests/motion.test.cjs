const {test}=require('node:test');
const assert=require('node:assert/strict');
const vm=require('node:vm');
const fs=require('node:fs');
const path=require('node:path');
const source=fs.readFileSync(path.join(__dirname,'../web/uno/motion.js'),'utf8');

function fixture(){
 const handlers={};let played=null;
 const makeCard=(id,x,playable)=>{
  const classes=new Set();
  const el={dataset:{card:id,playable:String(playable)},style:{setProperty(){}},classList:{add:c=>classes.add(c),remove:c=>classes.delete(c)},getBoundingClientRect:()=>({left:x,top:100,width:90,height:135}),getAttribute:()=>id,closest:()=>el,focus(){},classes};return el;
 };
 const cards=[makeCard('one',0,true),makeCard('two',40,false),makeCard('three',80,true)];
 const button={disabled:true,addEventListener:(name,fn)=>handlers['button:'+name]=fn};
 const label={textContent:''};
 const hand={children:cards,style:{setProperty(){}},scrollLeft:0,querySelectorAll:()=>cards,getBoundingClientRect:()=>({left:0,right:180}),addEventListener:(name,fn)=>handlers[name]=fn,setPointerCapture(){},releasePointerCapture(){}};
 const document={querySelector:s=>s==='.hand'?hand:null,querySelectorAll:()=>[],getElementById:id=>id==='inspect-play'?button:label};
 const context={window:{},document,matchMedia:()=>({matches:false}),performance:{now:()=>1000},setTimeout};
 vm.runInNewContext(source,context);const motion=context.window.UnoMotion;motion.bindHand(id=>played=id);
 const event=(card,x,extra={})=>({pointerType:'touch',pointerId:1,target:card,clientX:x,clientY:120,...extra});
 return {motion,cards,handlers,button,label,event,get played(){return played}};
}

test('Touch inspects a card without playing it, explicit button plays',()=>{
 const f=fixture();f.handlers.pointerdown(f.event(f.cards[0],10));f.handlers.pointerup(f.event(f.cards[0],10));
 assert.equal(f.played,null);assert.equal(f.button.disabled,false);assert.equal(f.label.textContent,'one');
 let suppressed=false;f.handlers.click({preventDefault(){suppressed=true},stopImmediatePropagation(){}});assert.equal(suppressed,true);
 f.handlers['button:click']();assert.equal(f.played,'one');
});
test('Sliding across overlapping cards reveals the next card without playing',()=>{
 const f=fixture();f.handlers.pointerdown(f.event(f.cards[0],10));f.handlers.pointermove(f.event(f.cards[0],95));f.handlers.pointerup(f.event(f.cards[0],95));
 assert.equal(f.label.textContent,'three');assert.equal(f.played,null);assert.ok(f.cards[2].classes.has('inspecting'));assert.ok(!f.cards[0].classes.has('inspecting'));
});
test('An unplayable card can be inspected but not played',()=>{
 const f=fixture();f.handlers.pointerdown(f.event(f.cards[1],50));f.handlers.pointerup(f.event(f.cards[1],50));
 assert.equal(f.label.textContent,'two');assert.equal(f.button.disabled,true);f.handlers['button:click']();assert.equal(f.played,null);
});
test('Cancelled touch clears preview and cannot play',()=>{
 const f=fixture();f.handlers.pointerdown(f.event(f.cards[0],10));f.handlers.pointercancel();assert.equal(f.button.disabled,true);f.handlers['button:click']();assert.equal(f.played,null);
});
test('Discard positions are deterministic and show public cards in order',()=>{
 const f=fixture();const cards=[{id:'public-one'},{id:'public-two'},{id:'public-three'}];const draw=c=>`<b>${c.id}</b>`;
 const one=f.motion.discardHTML(cards,false,draw),two=f.motion.discardHTML(cards,false,draw);
 assert.equal(one,two);assert.ok(one.includes('--pile-angle:'));assert.ok(one.includes('--pile-x:'));assert.ok(one.indexOf('public-one')<one.indexOf('public-three'));
});

test('Touch selection ignores synthetic mouse hover and retains only one lifted card',()=>{
 const f=fixture();f.handlers.pointerdown(f.event(f.cards[0],10));f.handlers.pointermove(f.event(f.cards[0],95));f.handlers.pointerup(f.event(f.cards[0],95));
 f.handlers.pointermove(f.event(f.cards[0],10,{pointerType:'mouse'}));
 assert.equal(f.label.textContent,'three');assert.equal(f.cards.filter(c=>c.classes.has('inspecting')).length,1);
 f.handlers.pointerleave({pointerType:'mouse'});assert.equal(f.label.textContent,'three');
});
test('Focus on a different card during a touch drag cannot create a second selection',()=>{
 const f=fixture();f.handlers.pointerdown(f.event(f.cards[0],10));f.handlers.pointermove(f.event(f.cards[0],95));
 f.handlers.focusin({target:f.cards[0]});
 assert.ok(f.cards[2].classes.has('inspecting'));assert.equal(f.cards.filter(c=>c.classes.has('inspecting')).length,1);
});
