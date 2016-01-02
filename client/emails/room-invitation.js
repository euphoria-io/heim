import React from 'react'

import { Item, Span, A } from 'react-html-email'
import { StandardEmail, TopBubbleBox, BodyBox, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo-active.png">
      <Item align="center">
        <Span {...textDefaults} fontSize={18}><strong>{'{{.SenderName}}'}</strong> invites you to join</Span>
      </Item>
      <Item align="center">
        <A href="{{.RoomURL}}">
          <Span {...textDefaults} fontSize={28} color={null}>&{'{{.RoomName}}'}</Span>
        </A>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item align="center">
        <Span {...textDefaults} color="#7d7d7d">A note from {'{{.SenderName}}'}:</Span>
      </Item>
      <Item>
        <Span {...textDefaults}>{'{{.SenderMessage}}'}</Span>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
