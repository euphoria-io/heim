import fs from 'fs'
import React from 'react'


const scriptSrc = fs.readFileSync(__dirname + '/../../build/heim/fast-touch.js', 'utf8')

export default function FastTouchScript() {
  return (
    <script dangerouslySetInnerHTML={{__html: scriptSrc}} />
  )
}
