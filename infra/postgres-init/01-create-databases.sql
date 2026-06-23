-- Cada serviço é dono exclusivo do seu database — nenhum deles faz
-- JOIN ou consulta direta no schema do outro. Isso é o "database per
-- service" aplicado no nível lógico (ver docs/01-CONTEXTO-PROJETO.md,
-- secao "Sobre os bancos - logical, not physical").

CREATE DATABASE ingestion_db;    -- telemetry-ingestion (Go)
CREATE DATABASE prediction_db;   -- irrigation-prediction (Python)
CREATE DATABASE management_db;   -- farm-management (C#) - Core Domain
