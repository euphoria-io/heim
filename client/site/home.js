import React from 'react'

import { MainPage, FancyLogo, FauxMessage } from './common'
import heimURL from '../lib/heimURL'


module.exports = (
  <MainPage title="euphoria!" className="welcome">
    <div className="splash">
      <FancyLogo />
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
          <a className="start-chatting big-green-button" href={heimURL('/room/welcome/')} target="_blank">come say hello!</a>
        </div>
      </div>
    </div>
  </MainPage>
)
