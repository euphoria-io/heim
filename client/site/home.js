import React from 'react'

import { MainPage, FancyLogo } from './common'
import heimURL from '../lib/heimURL'


module.exports = (
  <MainPage title="euphoria!" className="welcome" noDonation bottom={<div className="donation-banner"><a href="https://patreon.com/euphoriachat">Support us!</a></div>}>
    <div className="splash">
      <FancyLogo />
      <div className="info-box">
        <div className="description">
          <p>
            We're building a platform for cozy realtime discussion spaces.
          </p>
          <p>
            It's like a mix of chat, forums, and mailing lists, with your friends, organizations, and people around the world.
          </p>
          <a className="start-chatting big-green-button" href={heimURL('/room/welcome/')} target="_blank">come say hello!</a>
        </div>
      </div>
    </div>
  </MainPage>
)
