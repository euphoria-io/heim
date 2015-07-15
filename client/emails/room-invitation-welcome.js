// ignore Text, Image redefinition
// jshint -W079

var React = require('react')

var email = require('./email')
var Item = email.Item
var Text = email.Text
var Link = email.Link
var common = require('./common')
var textDefaults = common.textDefaults


module.exports = (
  <common.StandardEmail>
    <common.TopBubbleBox logo="logo.png">
      <Item align="center">
        <Text {...textDefaults} fontSize={18}>hi! <strong>{'{{.SenderName}}'}</strong> invites you to join a {'{{.RoomPrivacy}}'} chat room:</Text>
      </Item>
      <Item align="center">
        <Link href="https://euphoria.io/room/space">
          <Text {...textDefaults} fontSize={32} color={null}>&{'{{.RoomName}}'}</Text>
        </Link>
      </Item>
    </common.TopBubbleBox>
    <common.BodyBox>
      <Item align="center">
        <Text {...textDefaults} color="#7d7d7d">a note from @{'{{.SenderName}}'}:</Text>
      </Item>
      <Item>
        <Text {...textDefaults}>{'{{.SenderMessage}}'}</Text>
      </Item>
    </common.BodyBox>
    <common.BodyBox>
      <Item>
        <Text {...textDefaults}><Link href="{{.RoomURL}}">&{'{{.RoomName}}'}</Link> is hosted on <Link {...textDefaults} href="{{.SiteURL}}">{'{{.SiteName}}'}</Link>, a free online discussion platform. you don't have to sign up to chat &ndash; just click the link, enter a nickname, and you'll be chatting with {'{{.SenderName}}'} in moments.</Text>
      </Item>
    </common.BodyBox>
    <common.Footer>
      {common.standardFooter}
    </common.Footer>
  </common.StandardEmail>
)
