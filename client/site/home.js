var React = require('react')

var common = require('./common')


module.exports = (
  <common.MainPage title="euphoria!" className="welcome">
    <div className="splash">
      <div className="clicky">
        <a className="logo" href={common.heimURL('/room/welcome/')} tabIndex={1}>welcome</a>
        <div className="colors">
          <div className="a"></div>
          <div className="b"></div>
          <div className="c"></div>
          <div className="d"></div>
          <div className="e"></div>
        </div>
      </div>
      <h1>let's make the internet<br /> feel like home again.</h1>
      <div className="info-box">
        <div className="description">
          <div className="messages">
            <common.FauxMessage sender="euphoria" message="we're building a platform for cozy real time discussion spaces" />
            <common.FauxMessage sender="euphoria" message="it's like a mix of chat, forums, and mailing lists">
              <div className="replies">
                <common.FauxMessage sender="euphoria" message="with your friends, organizations, and people around the world." />
              </div>
            </common.FauxMessage>
          </div>
          <a className="start-chatting" href={common.heimURL('/room/welcome/')} target="_blank">come check it out. say hello!</a>
        </div>
        <ul className="features">
          <li className="chat">
            <div className="inner">
              <h2>chat for free</h2>
              <p>on your computer and phone</p>
            </div>
          </li>
          <li className="instant">
            <div className="inner">
              <h2>join instantly</h2>
              <p>no install or sign-up required</p>
            </div>
          </li>
        </ul>
      </div>
    </div>
  </common.MainPage>
)
