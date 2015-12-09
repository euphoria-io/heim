import React from 'react'

import heimURL from '../../lib/heimURL'


export default function Header() {
  return (
    <header>
      <div className="container">
        <a className="logo" href={heimURL('/')}>euphoria</a>
        <a className="whats-euphoria outline-button" href={heimURL('/about')}><span className="long">what's euphoria</span>?</a>
        <a className="start-chatting green-button" href={heimURL('/room/welcome/')} target="_blank">start chatting &raquo;</a>
      </div>
    </header>
  )
}
