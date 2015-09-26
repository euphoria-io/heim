var hueHash = require('./hue-hash')

/**
 * Determines if name contains the characters in the partial name, in order.
 */
module.exports.containsSubseq = function(name, part) {
  // Walk the characters in partial name, skipping forward in full
  // name to match until we can't find any more matches or we finish
  // walking.
  var offset = 0
  for (var partOffset = 0; partOffset < part.length; partOffset++) {
    var nextChar = part[partOffset]
    offset = name.indexOf(nextChar, offset)
    if (offset === -1) {
      return false
    }
    offset++
  }
  return true
}

/**
 * From a name and a partial produce a score. Scores are arrays of
 * constant length and are intended to be compared lexicographically.
 * Lower values are better matches. If the result is null, there is
 * no match whatsoever.
 */
module.exports.scoreMatch = function(name, part) {
  // FIXME Use proper Unicode-aware case-folding, if not already
  var partLowercase = part.toLowerCase()
  var nameLowercase = name.toLowerCase()
  // Check prefixes, then infixes, then subsequences -- and for
  // each, try case-sensitive and then insensitive.
  // Want something faster but uglier?
  // https://github.com/timmc/lib-1666/commit/6bd6f8a7635074f098e3d498cdd248450559b013
  var indexOfCs = name.indexOf(part);
  var indexOfCi = nameLowercase.indexOf(partLowercase);
  if (indexOfCs === 0) {
    return [-31, 0]
  } else if (indexOfCi === 0) {
    return [-30, 0]
  } else if (indexOfCs >= 0) {
    return [-21, indexOfCs]
  } else if (indexOfCi >= 0) {
    return [-20, indexOfCi]
  } else if (module.exports.containsSubseq(name, part)) {
    return [-11, 0]
  } else if (module.exports.containsSubseq(nameLowercase, partLowercase)) {
    return [-10, 0]
  } else {
    return null
  }
}

/**
 * Yield {completion, score} for a pair of name, stripped
 * partial name.
 */
function annotateScore(name, partStrip) {
  var stripped = hueHash.stripSpaces(name)
  var score = module.exports.scoreMatch(stripped, partStrip)
  if (score) {
    // Add tie-breakers. We first sort by lowercased names and
    // then by the original names so that we don't get orderings
    // like ["A", "Z", "a"]. This still sorts uppercase before
    // lowercase, which is fine.
    score.push(name.toLowerCase(), name)
  }
  return {completion: stripped, score: score}
}

/**
 * Custom lexicographic comparator on pairs of equal-length
 * arrays. Does not do a deep comparison on sub-arrays.
 */
function compareArrays(a, b) {
  var len = a.length;
  for (var i = 0; i < len; i++) {
    var elA = a[i]
    var elB = b[i]
    if (elA < elB) {
      return -1
    } else if (elA > elB) {
      return 1
    }
    // continue if equal...
  }
  return 0;
}

/**
 * Given an Immutable Seq of names and a partial name, yield sorted
 * Seq of mentionable names by match relevancy (best first).
 * Names that do not match at all are omitted from the result.
 * Mentionable names are suitable for use as mentions (do not contain
 * spaces, but do contain emoji, non-ASCII, etc.)
 */
module.exports.rankCompletions = function(names, part) {
  var partStrip = hueHash.stripSpaces(part)
  return names
    .filter(Boolean)
    .map(name => annotateScore(name, partStrip))
    .filter(entry => entry.score)
    // Use a custom lexicographic array sorter because JS's native
    // array comparison stringifies numeric elements for comparison,
    // meaning negative numbers compare incorrectly. We need negative
    // numbers in the score so that better matches come up at the
    // front -- because that needs to match the tie-breaker of
    // asciibetical ordering!
    .sortBy(entry => entry.score, compareArrays)
    .map(entry => entry.completion)
}
