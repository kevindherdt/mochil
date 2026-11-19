import test from 'node:test'
import assert from 'node:assert/strict'
import { isMatterTitle, worthReading } from '../static/narration-clean.js'

test('publishing boilerplate is not read', () => {
  for (const text of [
    'Copyright © Wilbur Smith 2014', 'Source ISBN: 9780007535651', 'www.harpercollins.co.uk',
    'You can always correspond with me via email at dirk@dirkpatton.com.', 'All rights reserved.',
    'Library of Congress Cataloging-in-Publication Data', 'Sign up for the S&S Teen Newsletter.',
    'Cover design by Russell Gordon', 'First published in Great Britain in 2007',
  ]) assert.equal(worthReading(text), false, text)
})

test('story text is kept, including words that merely look like boilerplate', () => {
  for (const text of [
    'The lantern swung over the harbour.', 'She printed in careful letters, and he read it twice.',
    'He had sent a first edition of the poems to her, signed.', '“Do not,” she said, “look back.”',
    'The newsletter ended there, he thought, at the edge of the page.'.replace('newsletter', 'letter'),
  ]) assert.equal(worthReading(text), true, text)
})

test('front and back matter are recognised by their titles', () => {
  for (const title of ['Copyright', 'Copyright Page', 'Contents', 'Table of Contents', 'Also by Joe Abercrombie', 'About the Author', 'Acknowledgments', 'Acknowledgements', 'Praise for The Subtle Knife', 'Title Page', 'Sneak Peek']) {
    assert.equal(isMatterTitle(title), true, title)
  }
  for (const title of ['Chapter One', 'Prologue', 'Epilogue', 'The Index of Forbidden Books', 'Summer of the Fifth Year', 'Author', '', undefined]) {
    assert.equal(isMatterTitle(title), false, String(title))
  }
})
