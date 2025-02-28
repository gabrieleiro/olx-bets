-- no game channel set no round
INSERT INTO guilds(discord_id) VALUES ('827261239926980668');

-- game channel set and no round
INSERT INTO guilds(discord_id, game_channel_id) VALUES ('927261239926980667', '1290463177401962537');

-- game channel set and round
INSERT INTO
    guilds(discord_id, game_channel_id)
    VALUES ('127261239926980822', '1230463177401962456');
INSERT INTO
    olx_ads(id, title, price, location, image, category)
    VALUES ('1', 'Conjunto de mesas e cadeiras plásticas', '950', 'Salvador - BA', 'https://img.olx.com.br/thumbs500x360/53/534553738598605.jpg', 'Móveis'); 
INSERT INTO rounds(guild_id, ad_id) VALUES ('127261239926980822', '1');

-- no game channel set and round
INSERT INTO
    guilds(discord_id)
    VALUES ('666261239926980822');
INSERT INTO
    olx_ads(id, title, price, location, image, category)
    VALUES ('2', 'Poltrona em tecido', '250', 'Belém - PA', 'https://img.olx.com.br/thumbs500x360/45/457481359528842.jpg', 'Móveis'); 
INSERT INTO rounds(guild_id, ad_id) VALUES ('666261239926980822', '2');

