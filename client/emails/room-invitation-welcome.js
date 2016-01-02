import React from 'react'

import { Item, Span, A } from 'react-html-email'
import { StandardEmail, TopBubbleBox, BodyBox, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo.png">
      <Item align="center">
        <Span {...textDefaults} fontSize={18}>Hi! <strong>{'{{.SenderName}}'}</strong> invites you to join a {'{{.RoomPrivacy}}'} chat room:</Span>
      </Item>
      <Item align="center">
        <A href="https://euphoria.io/room/space">
          <Span {...textDefaults} fontSize={28} color={null}>&{'{{.RoomName}}'}</Span>
        </A>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item align="center">
        <Span {...textDefaults} color="#7d7d7d">A note from @{'{{.SenderName}}'}:</Span>
      </Item>
      <Item>
        <Span {...textDefaults}>{'{{.SenderMessage}}'}</Span>
      </Item>
    </BodyBox>
    <BodyBox>
      <Item>
        <Span {...textDefaults}><A href="{{.RoomURL}}">&{'{{.RoomName}}'}</A> is hosted on <A {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</A>, a free online discussion platform. You don't have to sign up to chat &ndash; just click the link, enter a nickname, and you'll be chatting with {'{{.SenderName}}'} in moments.</Span>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
