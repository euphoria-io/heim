import React from 'react'

import heimURL from '../../lib/heim-url'


export default React.createClass({
  render() {
    return (
      <header>
        <div className="container">
          <a className="logo" href={heimURL('/')}>euphoria</a>
          <a className="whats-euphoria" href={heimURL('/about')}><span className="long">what's euphoria</span>?</a>
          <a className="start-chatting" href={heimURL('/room/welcome/')} target="_blank">start chatting &raquo;</a>
        </div>
      </header>
    )
  },
})

