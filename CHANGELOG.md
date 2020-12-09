# Changelog

# v0.25.0

- Added `!myevents` command to show events you are signed up for
- Added details to the `!admin announce` content for number of existing signups, if any
- Added the ability to custom sort the roles for an event instead of it always being alphabetical (still defaults to alphabetical)

## v0.24.0

- Enabled support for custom server emojis in reaction signups

## v0.23.0

- Signup/Withdraw by reaction is now possible
- Adjusted the `!show` formatting to make emojis always visible if enabled (previously they were only visible for signed-up roles)

## v0.22.0

- Start creating reactions in preparation for signup-by-reaction

## v0.21.0

- Switch to using replies instead of mentions

## v0.20.1

- Bugfixes with json deserialization

## v0.20.0

- Updated underlying libraries to deal with discord API changes

## v0.19.0

- Adjust bot permissions (with new lib version) to ensure ability to embed/attach

## v0.18.1 -- v0.18.4

- Get debug handlers actually working

## v0.18.0

- Add debug handlers to help figure out issues

## v0.17.1

- Upgrade libraries to get some moderately better logging (hopefully)

## v0.17.0

- Upgrade to bot-lib v10 and go-util v5

## v0.16.1

- Add guild_id to traces

## v0.16.0

- Add tracing and rework the service to ship logs off box

## v0.15.1

- Fix nil pointer dereference panic
- Use the bugfix version of the bot library
- Attempt to fix integer overflow issues that might already exist in the database

## v0.15.0

- Add bugsnag panic reporting

## v0.14.0

- Upgrade to the v7 bot library and v3 go-util

## v0.13.3

- Convert to go modules, new linters, and fix linting issues

## v0.13.2

- Fix a help placeholder for config-su

## v0.13.1

- Some refactoring to make the !admin signup and !admin withdraw actually work in the signup channel

## v0.13.0

- Allow !admin signup and !admin withdraw in signup channels as well as admin channels

## v0.12.1

- Fix `!admin edit` creating new events unintentionally
- Improve logging for error diagnosis

## v0.12.0

- Add the ability to signup for multiple events with `!su Event1 Role1 Event2 Role2`
- Attempt to understand more than just straight-quotes (") when tokenizing. Now treat ", “, ”, «, », and „ equivalently.

## v0.11.2

- Make the bot aware of channel creations and renames more reliably
- Fix the `!admin grouping` command output
- Fix some other crash-causing issues

## v0.11.1

- Fix `!admin edit trial announceto=...`
- Add `!admin show trialname`

## v0.11.0

- Add `!admin clear [trialname]`
- s/trial/event/ in (at least some) places that it can be seen by users
- Enable normal commands working in the admin channel when used by admin users
- Add `!config-su website`, `!config-su discord`, and `!config-su stats`
- Changed the `!admin grouping` output to only mention those signed up and not display a full layout

## v0.10.0

- Add AdminRole functionality and proper AdminChannel filtering

## v0.9.5

- Make !list work when there are no events
- Internal improvements

## v0.9.4

- No longer respond to unknown commands by default (plays better with other bots)

## v0.9.3

- More improved logging

## v0.9.2

- Internal cleanups and improved logging

## v0.9.1

- Fix issue with removing roles from a trial

## v0.9.0

- Add ability to have emoji with role displays
- Fix multiple-signups for the same role

## v0.8.0

- Add multi-signup and multi-withdraw for admin
- Make admin respect the ShowAfterSignup and ShowAfterWithdraw

## v0.7.1

- Enable config version command instead of common one

## v0.7.0

- Add dump binary
- Fix inconsistent admin signup/withdraw
- Add logging
- Restrict signups to signup channel

## v0.6.0

- Arrange for case-insensitive commands by default

## v0.5.0

- Add some more settings for smoother operations

## v0.4.0

- Add grouping, signup, and withdraw admin commands
- Add validation that required mentions are in fact mentions

## v0.3.5

- Fix withdraw bug

## v0.3.4

- Fix some bugs

## v0.3.3

- Make !list channel references work

## v0.3.2

- Fix role list in announce and channel references in list

## v0.3.1

- GetTrial can find legacy trials now too

## v0.3.0

- Some refactoring for underlying library upgrade
- Enable admin announce, admin delete

## v0.2.0

- Fix "show"
- Add `su` as an alias for `signup`
- Add `wd` as an alias for `withdraw`

## v0.1.6

- First pass at an almost-working bot
- Enable message colors