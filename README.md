# navirpc

displays navidrome now-playing onto your discord profile via rich presence: track, artist, album art, live progress bar. headless, no discord desktop client, no account token.

it sets presence through discord's social-layer headless-session api on a real oauth grant.

this is entirely proof of concept. it runs end to end on a real instance and has since early july, but alot of the intended features i want aren't ready, it is not packaged, not released, and carries no versioned build yet.

`web/` is a page that proves the 100% local browser oauth flow, it's bare and ugly and hardcodes a central app id right now, but i'm working on a more published page that'll be deployed

## how it works

- navidrome's scrobbler capability delivers playback reports to the plugin
- the plugin maps a report to a discord activity and posts it to discord at `/users/@me/headless-sessions`
- a scheduler tick keeps the session alive against discord's 20 minute TTL and flushes anything the rate throttle deferred
- each user's connect token lives in the plugin's kv-store and refreshes itself before expiry

## authenticating with discord

worth clearing up first, because "discord app" means two different things. everywhere below it means a **discord developer application**, the thing you register in the developer portal to get a client id, the same object sitting behind every "log in with discord" button you've ever clicked. it is not the discord client you chat in. navirpc never talks to that and you never need it installed, let alone running.

the application is an identity, not a service. it holds no token, and there's no server of mine anywhere in this.

you open the page, it bounces you to discord's consent screen, discord sends you back with a code, and the page swaps that code for a token in your own browser. discord's token endpoint allows CORS so there genuinely is no backend, and PKCE is what makes that safe without a client secret sitting in public javascript.

what comes back is a refresh token. paste it into the plugin config next to your navidrome username and the setup is done. the plugin keeps it in the kv-store and trades it for a short lived access token whenever it needs one. discord invalidates the old refresh token every time it issues a new one, so the plugin persists the new one before it uses anything, otherwise a crash at the wrong moment strands the account on a token discord has already killed.

the scope is `sdk.social_layer_presence`, which is presence writes plus reading your own profile and your friends. no messages, no email, no account access. if one leaks, someone can set your status and see who you're friends with, and you revoke it from the authorised apps list in your own discord settings.

the application has to have the **Social SDK enabled** in the developer portal or authorize just returns `invalid_scope`, and the scope still shows as tickable in the oauth2 url generator either way. that one cost me an afternoon.

the plan is that a central application is the default, so the normal path is a login and a paste with nothing to register and no portal to visit. anyone who'd rather make their own application themselves can point it at that instead, and it genuinely makes no difference to what you get, same card, same scope, same behaviour, the only thing that changes is whose client id minted your token. the plugin binds each user to whichever application issued theirs, multiple can coexist on one instance. worth knowing refresh tokens are bound to the application that issued them, so moving between the two means reconnecting.

right now the poc just hardcodes the central id in the page. using your own comes with the proper tool.

## caveats

- **tokens are plaintext at rest** in the plugin's kv-store. the wasm sandbox has no host keystore, so at-rest encryption would only move the key problem somewhere else. keep the navidrome data directory permission restricted.

- **discords rate limits are super anal** 5 calls max every 20 seconds, and the plugin holds itself to 4 to stay clear of it. genuinely fine for pushing what navidrome tracks (play, pause, track changes) but any rapid back and forth seeks will throttle the calls.


## polish & release

i plan to have this polished & released with the full tool and proper documentation within the next couple weeks, bear with me