-- Реальные платежи через CloudPayments: откуда платёж и его id у провайдера.
-- UNIQUE на provider_tx_id делает обработку уведомлений идемпотентной (повтор не создаст дубль).
ALTER TABLE payments ADD COLUMN IF NOT EXISTS provider TEXT NOT NULL DEFAULT 'test';
ALTER TABLE payments ADD COLUMN IF NOT EXISTS provider_tx_id TEXT UNIQUE;
