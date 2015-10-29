import React from 'react'


export function render(pageComponent) {
  const doctype = '<!doctype html>'
  return doctype + React.renderToStaticMarkup(pageComponent)
}
