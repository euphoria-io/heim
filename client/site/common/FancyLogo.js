import React from 'react'

import heimURL from '../../lib/heim-url'


export default function FancyLogo() {
  return (
    <div className="fancy-logo">
      <a className="logo" href={heimURL('/room/welcome/')} tabIndex={1}>welcome</a>
      <div className="colors">
        <div className="a"></div>
        <div className="b"></div>
        <div className="c"></div>
        <div className="d"></div>
        <div className="e"></div>
      </div>
    </div>
  )
}
