import React from 'react'

import { Item, Text, Link } from './email'
import { StandardEmail, TopBubbleBox, BodyBox, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo-active.png">
      <Item align="center">
        <Text {...textDefaults} fontSize={18}><strong>{'{{.SenderName}}'}</strong> invites you to join</Text>
      </Item>
      <Item align="center">
        <Link href="{{.RoomURL}}">
          <Text {...textDefaults} fontSize={28} color={null}>&{'{{.RoomName}}'}</Text>
        </Link>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item align="center">
        <Text {...textDefaults} color="#7d7d7d">A note from {'{{.SenderName}}'}:</Text>
      </Item>
      <Item>
        <Text {...textDefaults}>{'{{.SenderMessage}}'}</Text>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
