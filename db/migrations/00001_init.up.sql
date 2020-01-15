CREATE TABLE IF NOT EXISTS photos (
    id SERIAL NOT NULL PRIMARY KEY,
    image_hash BYTEA,
    caption TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    photo_id BIGINT NOT NULL,
    media_album_id BIGINT NOT NULL,
    file_id BIGINT NOT NULL,
    sender_user_id BIGINT NOT NULL,
    is_downloading_active BOOLEAN NOT NULL,
    is_downloading_completed BOOLEAN NOT NULL,
    is_uploading_active BOOLEAN NOT NULL,
    is_uploading_completed BOOLEAN NOT NULL,
    file_path TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    published_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_photos_image_hash ON photos (image_hash);
CREATE UNIQUE INDEX IF NOT EXISTS idx_photos_message_id ON photos (chat_id, message_id);
CREATE INDEX IF NOT EXISTS idx_photos_published_at ON photos (published_at);

CREATE TABLE IF NOT EXISTS videos (
    id SERIAL NOT NULL PRIMARY KEY,
    caption TEXT NOT NULL,
    chat_id BIGINT NOT NULL,
    message_id BIGINT NOT NULL,
    media_album_id BIGINT NOT NULL,
    file_id BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    sender_user_id BIGINT NOT NULL,
    is_downloading_active BOOLEAN NOT NULL,
    is_downloading_completed BOOLEAN NOT NULL,
    is_uploading_active BOOLEAN NOT NULL,
    is_uploading_completed BOOLEAN NOT NULL,
    file_path TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    published_at INTEGER NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_message_id ON videos (chat_id, message_id);
CREATE INDEX IF NOT EXISTS idx_videos_published_at ON videos (published_at);

CREATE TABLE IF NOT EXISTS chats (
    id SERIAL NOT NULL PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    title TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_chats_chat_id ON chats (chat_id);

CREATE TABLE IF NOT EXISTS messages (
    message_type SMALLINT NOT NULL,
    message_id BIGINT NOT NULL,
    chat_id BIGINT NOT NULL,
    media_album_id BIGINT NOT NULL,
    uploaded BOOLEAN NOT NULL,
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,
    published_at INTEGER NOT NULL,
    PRIMARY KEY (chat_id,message_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_message_id ON messages (chat_id, message_id);
CREATE INDEX IF NOT EXISTS id_messages_uploaded ON messages (uploaded);