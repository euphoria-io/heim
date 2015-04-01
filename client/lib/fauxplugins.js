var React = require('react')


module.exports = function(roomName) {
  if (roomName == 'thedrawingroom' || roomName == 'lovenest') {
    Heim.hook('page-bottom', function() {
      return (
        <style key="drawingroom-style" dangerouslySetInnerHTML={{__html:`
          .chat {
            background: #333;
          }

          .chat .room .name {
            color: #222;
          }

          .chat time {
            opacity: .5;
          }

          .chat .room .privacy-level,
          .nick {
            background: #e8e8e8 !important;
          }

          .message-emote {
            background: #f3f3f3 !important;
          }

          .mention-nick {
            color: #000 !important;
            font-weight: bold;
          }

          a {
            color: #444;
            text-decoration: none;
            font-weight: bold;
          }
        `}} />
      )
    })
  }

  if (roomName == 'space') {
    Heim.hook('sidebar', function() {
      return (
        <div key="norman" className="norman">
          <p>norman</p>
          <img src="//i.imgur.com/UKbitCO.jpg" />
        </div>
      )
    })

    Heim.hook('page-bottom', function() {
      return (
        <style key="norman-style" dangerouslySetInnerHTML={{__html:`
          .norman {
            margin-top: 15px;
            text-align: right;
            opacity: .5;
          }

          .norman, .norman img {
            transition: all .15s ease;
          }

          .norman:hover {
            opacity: 1;
          }

          .norman:hover img {
            width: 22vw;
            max-width: 250px;
          }

          .norman p {
            margin: 0;
            font-size: 12px;
          }

          .norman img {
            width: 15vw;
            min-width: 50px;
            max-width: 100px;
          }
        `}} />
      )
    })
  }

  if (roomName == 'mantodea') {
    var _ = require('lodash')

    var serverTime = Number.MAX_VALUE
    var cutoff = 7200
    var cutoffReached = false

    Heim.socket.store.listen(function(ev) {
      if (ev.status == 'receive' && ev.body.type == 'ping-event') {
        serverTime = ev.body.data.time
      }
    })

    Heim.hook('incoming-messages', function(messages) {
      _.remove(messages, function(msg) {
        var tooOld = msg.time < serverTime - cutoff
        cutoffReached = cutoffReached || tooOld
        return tooOld
      })
    })

    Heim.actions.loadMoreLogs.shouldEmit = function() {
      return !cutoffReached
    }
  }

  Heim.addEventListener(uidocument.body, 'keydown', function(ev) {
    var SyntheticKeyboardEvent = require('react/lib/SyntheticKeyboardEvent')
    var reactEvent = new SyntheticKeyboardEvent(null, null, ev)
    if (reactEvent.key == 'Backspace' || reactEvent.key == 'Delete') {
      reactEvent.preventDefault()
      reactEvent.stopPropagation()
    }
  }, true)
}
