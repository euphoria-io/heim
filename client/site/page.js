import ReactDOMServer from 'react-dom/server'


export function render(pageComponent) {
  const doctype = '<!doctype html>'
  return doctype + ReactDOMServer.renderToStaticMarkup(pageComponent)
}
