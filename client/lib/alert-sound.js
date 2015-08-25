var sound = new Audio(process.env.HEIM_PREFIX + '/static/alert.mp3')

module.exports = {}
module.exports.play = function() {
  if (sound.readyState !== 0) {
    sound.currentTime = 0
  }
  sound.play()
}
