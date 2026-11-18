const directions = new Set(['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'])

export function spatialTargetIndex(rects, currentIndex, direction) {
  const current=rects[currentIndex]
  if(!current||!directions.has(direction))return -1
  const horizontal=direction==='ArrowLeft'||direction==='ArrowRight'
  const forward=direction==='ArrowRight'||direction==='ArrowDown'
  const cx=(current.left+current.right)/2,cy=(current.top+current.bottom)/2
  let best=-1,bestScore=Infinity
  rects.forEach((rect,index)=>{
    if(index===currentIndex)return
    const x=(rect.left+rect.right)/2,y=(rect.top+rect.bottom)/2
    const primary=horizontal?x-cx:y-cy
    if(forward?primary<=1:primary>=-1)return
    const secondary=Math.abs(horizontal?y-cy:x-cx)
    const overlap=horizontal
      ? Math.max(0,Math.min(current.bottom,rect.bottom)-Math.max(current.top,rect.top))
      : Math.max(0,Math.min(current.right,rect.right)-Math.max(current.left,rect.left))
    const score=Math.abs(primary)+(secondary*3)-(overlap*2)
    if(score<bestScore){best=index;bestScore=score}
  })
  return best
}

/** Adds remote-like arrow navigation without changing normal Tab order. */
export function createSpatialNavigation({container,selector}) {
  const candidates=()=>[...container.querySelectorAll(selector)].filter(element=>!element.disabled&&element.getClientRects().length)
  const onKeyDown=event=>{
    if(!directions.has(event.key)||event.altKey||event.ctrlKey||event.metaKey||event.shiftKey)return
    const items=candidates(),current=items.indexOf(event.target)
    if(current<0)return
    const target=spatialTargetIndex(items.map(item=>item.getBoundingClientRect()),current,event.key)
    if(target<0)return
    event.preventDefault();items[target].focus({preventScroll:true});items[target].scrollIntoView({block:'nearest',inline:'nearest'})
  }
  container.addEventListener('keydown',onKeyDown)
  return {destroy:()=>container.removeEventListener('keydown',onKeyDown)}
}
