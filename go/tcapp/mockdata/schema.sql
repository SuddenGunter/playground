CREATE TABLE IF NOT EXISTS mock_data (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name       TEXT        NOT NULL CONSTRAINT mock_data_name_length  CHECK (char_length(name)  <= 256),
    value      TEXT        NOT NULL CONSTRAINT mock_data_value_length CHECK (char_length(value) <= 4096),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
