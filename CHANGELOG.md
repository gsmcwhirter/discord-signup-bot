# Changelog

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