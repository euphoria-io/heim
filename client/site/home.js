import React from 'react'

import { MainPage, FancyLogo, FauxMessage } from './common'
import heimURL from '../lib/heim-url'


module.exports = (
  <MainPage title="euphoria!" className="welcome">
    <div className="splash">
      <FancyLogo />
      <h1>let's make the internet<br /> feel like home again.</h1>
      <div className="info-box">
        <div className="description">
          <div className="messages">
            <FauxMessage sender="euphoria" message="we're building a platform for cozy real time discussion spaces" />
            <FauxMessage sender="euphoria" message="it's like a mix of chat, forums, and mailing lists">
              <div className="replies">
                <FauxMessage sender="euphoria" message="with your friends, organizations, and people around the world." />
              </div>
            </FauxMessage>
          </div>
          <a className="start-chatting big-green-button" href={heimURL('/room/welcome/')} target="_blank">come check it out. say hello!</a>
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
  </MainPage>
)
