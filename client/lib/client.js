var React = require('react')
require('superagent-bluebird-promise')

var Main = require('./ui/main')


React.render(
  <Main />,
  document.getElementById('container')
)

ChatStore = require('./stores/chat')
ChatStore.trigger({
  messages: [
    {text: 'hello world'},
    {text: 'j0'},
    {text: 'j0!!!'},
  ]
})
