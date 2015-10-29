import React from 'react'


export default function renderEmail(emailComponent) {
  const doctype = '<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">'
  return doctype + React.renderToStaticMarkup(emailComponent)
}
