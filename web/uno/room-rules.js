window.RoomRules={
 secondsRemaining(deadline,serverNow,elapsed){return Math.max(0,Math.ceil((deadline-serverNow-elapsed)/1000))},
 formatSeconds(seconds){return `${Math.floor(seconds/60)}:${String(seconds%60).padStart(2,'0')}`},
 describe(o){return `${o.turnSeconds?o.turnSeconds+' s por turno':'Sin límite de tiempo'} · ${o.drawUntilPlayable?'Robar hasta poder jugar':'Robar una carta'} · ${o.forcePlay?'Jugada obligatoria':'Jugada voluntaria'} · ${o.mixDrawCards?'Mezclar +2 y +4':'Acumular solo el mismo tipo'}`}
};
