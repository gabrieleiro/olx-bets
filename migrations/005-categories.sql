CREATE TABLE disabled_categories (
    guild_id INTEGER NOT NULL,
    category TEXT NOT NULL,
    PRIMARY KEY (guild_id, category)
);

ALTER TABLE olx_ads ADD COLUMN category TEXT;
