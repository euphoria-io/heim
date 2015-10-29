import React from 'react'

import { Item, Text, Link } from './email'
import { StandardEmail, TopBubbleBox, BodyBox, standardFooter, textDefaults } from './common'


module.exports = (
  <StandardEmail>
    <TopBubbleBox logo="logo-warning.png" padding={15}>
      <Item align="center">
        <Text {...textDefaults} fontSize={24}>your password has been changed</Text>
      </Item>
    </TopBubbleBox>
    <BodyBox>
      <Item>
        <Text {...textDefaults}>hey {'{{.AccountName}}'}, just keeping you in the loop. if you just updated your <Link {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</Link> password, you're good to go!</Text>
      </Item>
      <Item>
        <Text {...textDefaults}>if you did not change your password and suspect something fishy is going on, please reply to this email immediately.</Text>
      </Item>
    </BodyBox>
    {standardFooter}
  </StandardEmail>
)
