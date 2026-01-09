--
-- PostgreSQL database dump
--


-- Dumped from database version 16.11 (Ubuntu 16.11-0ubuntu0.24.04.1)
-- Dumped by pg_dump version 16.11 (Ubuntu 16.11-0ubuntu0.24.04.1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: account_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.account_type AS ENUM (
    'demo',
    'live'
);


--
-- Name: emotion_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.emotion_type AS ENUM (
    'calm',
    'anxious',
    'fomo',
    'revenge',
    'other'
);


--
-- Name: execution_event_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.execution_event_type AS ENUM (
    'entry',
    'partial_close',
    'tp_hit',
    'sl_hit',
    'manual_close'
);


--
-- Name: rule_result_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.rule_result_type AS ENUM (
    'PASS',
    'FAIL'
);


--
-- Name: rule_timeframe; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.rule_timeframe AS ENUM (
    'W1'
);


--
-- Name: session_type; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.session_type AS ENUM (
    'london',
    'new_york',
    'asian',
    'custom'
);


--
-- Name: trade_bias; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.trade_bias AS ENUM (
    'long',
    'short'
);


--
-- Name: trade_result; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.trade_result AS ENUM (
    'win',
    'loss',
    'breakeven'
);


--
-- Name: prevent_duplicate_entry(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.prevent_duplicate_entry() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.event_type = 'entry' THEN
        IF EXISTS (
            SELECT 1 FROM trade_executions
            WHERE trade_id = NEW.trade_id
            AND event_type = 'entry'
        ) THEN
            RAISE EXCEPTION 'Cannot enter: trade % already has entry', NEW.trade_id;
        END IF;
    END IF;
    RETURN NEW;
END;
$$;


--
-- Name: prevent_execution_after_close(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.prevent_execution_after_close() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- Check if trade is already closed
    IF EXISTS (
        SELECT 1 FROM trade_executions
        WHERE trade_id = NEW.trade_id
        AND event_type IN ('tp_hit', 'sl_hit', 'manual_close')
    ) THEN
        RAISE EXCEPTION 'Cannot execute:  trade % is already closed', NEW.trade_id;
    END IF;
    RETURN NEW;
END;
$$;


--
-- Name: prevent_intent_after_execution(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.prevent_intent_after_execution() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    -- Check if trade has any executions
    IF EXISTS (
        SELECT 1 FROM trade_executions
        WHERE trade_id = NEW.trade_id
    ) THEN
        RAISE EXCEPTION 'Cannot set intent: trade % has already been executed', NEW.trade_id;
    END IF;
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: accounts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.accounts (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    type public.account_type NOT NULL,
    broker_name text NOT NULL,
    currency character(3) NOT NULL,
    balance numeric(15,2) NOT NULL,
    leverage integer NOT NULL,
    max_risk_per_trade_pct numeric(5,2) NOT NULL,
    max_daily_risk_pct numeric(5,2) NOT NULL,
    timezone text NOT NULL,
    preferred_session public.session_type DEFAULT 'london'::public.session_type NOT NULL,
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT accounts_balance_check CHECK ((balance >= (0)::numeric)),
    CONSTRAINT accounts_currency_check CHECK ((currency ~ '^[A-Z]{3}$'::text)),
    CONSTRAINT accounts_leverage_check CHECK ((leverage > 0)),
    CONSTRAINT accounts_max_daily_risk_pct_check CHECK (((max_daily_risk_pct >= (0)::numeric) AND (max_daily_risk_pct <= (100)::numeric))),
    CONSTRAINT accounts_max_risk_per_trade_pct_check CHECK (((max_risk_per_trade_pct >= (0)::numeric) AND (max_risk_per_trade_pct <= (100)::numeric))),
    CONSTRAINT accounts_timezone_check CHECK ((timezone ~ '^[^/]+(/[A-Za-z_/-]+)*$'::text))
);


--
-- Name: candles_weekly; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.candles_weekly (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    timestamp_utc timestamp with time zone NOT NULL,
    open numeric(12,5) NOT NULL,
    high numeric(12,5) NOT NULL,
    low numeric(12,5) NOT NULL,
    close numeric(12,5) NOT NULL,
    volume bigint,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT candles_weekly_check CHECK ((low <= high))
);


--
-- Name: indicators_weekly; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.indicators_weekly (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    candle_id uuid NOT NULL,
    ema20 numeric(12,5) NOT NULL,
    ema50 numeric(12,5) NOT NULL,
    ema200 numeric(12,5) NOT NULL,
    range_size numeric(12,5) NOT NULL,
    body_size numeric(12,5) NOT NULL,
    upper_wick numeric(12,5) NOT NULL,
    lower_wick numeric(12,5) NOT NULL,
    mid_price numeric(12,5) NOT NULL,
    last_swing_high_price numeric(12,5),
    last_swing_low_price numeric(12,5),
    computed_at timestamp with time zone DEFAULT now()
);


--
-- Name: rule_results; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rule_results (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    rule_id uuid NOT NULL,
    candle_id uuid NOT NULL,
    result public.rule_result_type NOT NULL,
    evaluated_at timestamp with time zone DEFAULT now(),
    confidence_score numeric(3,2)
);


--
-- Name: rules; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.rules (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    code text NOT NULL,
    name text NOT NULL,
    timeframe public.rule_timeframe DEFAULT 'W1'::public.rule_timeframe NOT NULL,
    description text NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: trade_executions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trade_executions (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    trade_id uuid NOT NULL,
    event_type public.execution_event_type NOT NULL,
    price numeric(12,5) NOT NULL,
    position_size numeric(12,8) NOT NULL,
    executed_at timestamp with time zone NOT NULL,
    session public.session_type,
    reason text,
    slippage_pips numeric(8,2),
    pnl numeric(12,2),
    pnl_pips numeric(12,2),
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT valid_execution_data CHECK (((price > (0)::numeric) AND (position_size > (0)::numeric)))
);


--
-- Name: TABLE trade_executions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.trade_executions IS 'Append-only execution event log.  Contains MARKET INTERACTIONS only.  State is computed via:  SELECT event_type FROM trade_executions WHERE trade_id = ?  ORDER BY executed_at';


--
-- Name: trade_feedback; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trade_feedback (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    trade_id uuid NOT NULL,
    followed_plan boolean NOT NULL,
    emotion_before public.emotion_type NOT NULL,
    emotion_during public.emotion_type NOT NULL,
    emotion_after public.emotion_type NOT NULL,
    biggest_mistake text,
    screenshot_url text,
    feedback_at timestamp with time zone DEFAULT now()
);


--
-- Name: trade_intents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trade_intents (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    trade_id uuid NOT NULL,
    intent_type character varying(20) NOT NULL,
    reason text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT trade_intents_intent_type_check CHECK (((intent_type)::text = ANY ((ARRAY['cancel'::character varying, 'invalidate'::character varying])::text[])))
);


--
-- Name: TABLE trade_intents; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.trade_intents IS 'Records user/system intent to cancel or invalidate trades. Separate from executions because these are NOT market interactions. ';


--
-- Name: trades; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.trades (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    account_id uuid NOT NULL,
    candle_id uuid NOT NULL,
    symbol text DEFAULT 'EURUSD'::text NOT NULL,
    timeframe text DEFAULT 'W1'::text NOT NULL,
    setup_timestamp_utc timestamp with time zone NOT NULL,
    account_balance_at_setup numeric(15,2) NOT NULL,
    leverage_at_setup integer NOT NULL,
    max_risk_per_trade_pct_at_setup numeric(5,2) NOT NULL,
    timezone_at_setup text NOT NULL,
    bias public.trade_bias NOT NULL,
    planned_entry numeric(12,5) NOT NULL,
    planned_sl numeric(12,5) NOT NULL,
    planned_tp numeric(12,5) NOT NULL,
    planned_rr numeric(5,2) NOT NULL,
    planned_risk_pct numeric(5,2) NOT NULL,
    planned_risk_amount numeric(15,2) NOT NULL,
    planned_position_size numeric(10,5) NOT NULL,
    reason_for_trade text NOT NULL,
    actual_entry numeric(12,5),
    actual_sl numeric(12,5),
    actual_tp numeric(12,5),
    actual_risk_pct numeric(5,2),
    actual_risk_amount numeric(15,2),
    actual_position_size numeric(10,5),
    execution_timestamp_utc timestamp with time zone,
    close_timestamp_utc timestamp with time zone,
    close_price numeric(12,5),
    result public.trade_result,
    pips_gained numeric(8,2),
    money_gained numeric(15,2),
    rr_realized numeric(5,2),
    duration_seconds integer,
    session public.session_type,
    created_at timestamp with time zone DEFAULT now(),
    CONSTRAINT trades_planned_rr_check CHECK ((planned_rr > (0)::numeric)),
    CONSTRAINT trades_symbol_check CHECK ((symbol = 'EURUSD'::text)),
    CONSTRAINT trades_timeframe_check CHECK ((timeframe = 'W1'::text))
);


--
-- Name: TABLE trades; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.trades IS 'Trade state derived from trade_executions and trade_intents. ';


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: accounts accounts_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_pkey PRIMARY KEY (id);


--
-- Name: candles_weekly candles_weekly_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candles_weekly
    ADD CONSTRAINT candles_weekly_pkey PRIMARY KEY (id);


--
-- Name: candles_weekly candles_weekly_timestamp_utc_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.candles_weekly
    ADD CONSTRAINT candles_weekly_timestamp_utc_key UNIQUE (timestamp_utc);


--
-- Name: indicators_weekly indicators_weekly_candle_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.indicators_weekly
    ADD CONSTRAINT indicators_weekly_candle_id_key UNIQUE (candle_id);


--
-- Name: indicators_weekly indicators_weekly_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.indicators_weekly
    ADD CONSTRAINT indicators_weekly_pkey PRIMARY KEY (id);


--
-- Name: rule_results rule_results_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rule_results
    ADD CONSTRAINT rule_results_pkey PRIMARY KEY (id);


--
-- Name: rule_results rule_results_rule_id_candle_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rule_results
    ADD CONSTRAINT rule_results_rule_id_candle_id_key UNIQUE (rule_id, candle_id);


--
-- Name: rules rules_code_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_code_key UNIQUE (code);


--
-- Name: rules rules_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rules
    ADD CONSTRAINT rules_pkey PRIMARY KEY (id);


--
-- Name: trade_executions trade_executions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_executions
    ADD CONSTRAINT trade_executions_pkey PRIMARY KEY (id);


--
-- Name: trade_feedback trade_feedback_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_feedback
    ADD CONSTRAINT trade_feedback_pkey PRIMARY KEY (id);


--
-- Name: trade_feedback trade_feedback_trade_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_feedback
    ADD CONSTRAINT trade_feedback_trade_id_key UNIQUE (trade_id);


--
-- Name: trade_intents trade_intents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_intents
    ADD CONSTRAINT trade_intents_pkey PRIMARY KEY (id);


--
-- Name: trades trades_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trades
    ADD CONSTRAINT trades_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_candles_timestamp; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_candles_timestamp ON public.candles_weekly USING btree (timestamp_utc);


--
-- Name: idx_executions_trade_time; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_executions_trade_time ON public.trade_executions USING btree (trade_id, executed_at);


--
-- Name: idx_rule_results_candle_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rule_results_candle_id ON public.rule_results USING btree (candle_id);


--
-- Name: idx_rule_results_rule_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_rule_results_rule_id ON public.rule_results USING btree (rule_id);


--
-- Name: idx_trade_executions_event_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trade_executions_event_type ON public.trade_executions USING btree (event_type);


--
-- Name: idx_trade_executions_executed_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trade_executions_executed_at ON public.trade_executions USING btree (executed_at);


--
-- Name: idx_trade_executions_trade_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trade_executions_trade_id ON public.trade_executions USING btree (trade_id);


--
-- Name: idx_trade_executions_unique_entry; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_trade_executions_unique_entry ON public.trade_executions USING btree (trade_id, event_type) WHERE (event_type = 'entry'::public.execution_event_type);


--
-- Name: idx_trade_feedback_trade_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trade_feedback_trade_id ON public.trade_feedback USING btree (trade_id);


--
-- Name: idx_trade_intents_unique; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX idx_trade_intents_unique ON public.trade_intents USING btree (trade_id);


--
-- Name: idx_trades_bias; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trades_bias ON public.trades USING btree (bias);


--
-- Name: idx_trades_candle_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trades_candle_id ON public.trades USING btree (candle_id);


--
-- Name: idx_trades_result; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trades_result ON public.trades USING btree (result);


--
-- Name: idx_trades_session; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trades_session ON public.trades USING btree (session);


--
-- Name: idx_trades_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_trades_user_id ON public.trades USING btree (user_id);


--
-- Name: uniq_trade_account_candle_bias; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX uniq_trade_account_candle_bias ON public.trades USING btree (account_id, candle_id, bias);


--
-- Name: trade_executions trg_prevent_duplicate_entry; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_prevent_duplicate_entry BEFORE INSERT ON public.trade_executions FOR EACH ROW EXECUTE FUNCTION public.prevent_duplicate_entry();


--
-- Name: trade_executions trg_prevent_execution_after_close; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_prevent_execution_after_close BEFORE INSERT ON public.trade_executions FOR EACH ROW EXECUTE FUNCTION public.prevent_execution_after_close();


--
-- Name: trade_intents trg_prevent_intent_after_execution; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_prevent_intent_after_execution BEFORE INSERT ON public.trade_intents FOR EACH ROW EXECUTE FUNCTION public.prevent_intent_after_execution();


--
-- Name: accounts accounts_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.accounts
    ADD CONSTRAINT accounts_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: indicators_weekly indicators_weekly_candle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.indicators_weekly
    ADD CONSTRAINT indicators_weekly_candle_id_fkey FOREIGN KEY (candle_id) REFERENCES public.candles_weekly(id) ON DELETE CASCADE;


--
-- Name: rule_results rule_results_candle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rule_results
    ADD CONSTRAINT rule_results_candle_id_fkey FOREIGN KEY (candle_id) REFERENCES public.candles_weekly(id) ON DELETE CASCADE;


--
-- Name: rule_results rule_results_rule_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.rule_results
    ADD CONSTRAINT rule_results_rule_id_fkey FOREIGN KEY (rule_id) REFERENCES public.rules(id);


--
-- Name: trade_executions trade_executions_trade_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_executions
    ADD CONSTRAINT trade_executions_trade_id_fkey FOREIGN KEY (trade_id) REFERENCES public.trades(id) ON DELETE CASCADE;


--
-- Name: trade_feedback trade_feedback_trade_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_feedback
    ADD CONSTRAINT trade_feedback_trade_id_fkey FOREIGN KEY (trade_id) REFERENCES public.trades(id) ON DELETE CASCADE;


--
-- Name: trade_intents trade_intents_trade_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trade_intents
    ADD CONSTRAINT trade_intents_trade_id_fkey FOREIGN KEY (trade_id) REFERENCES public.trades(id) ON DELETE CASCADE;


--
-- Name: trades trades_account_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trades
    ADD CONSTRAINT trades_account_id_fkey FOREIGN KEY (account_id) REFERENCES public.accounts(id);


--
-- Name: trades trades_candle_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trades
    ADD CONSTRAINT trades_candle_id_fkey FOREIGN KEY (candle_id) REFERENCES public.candles_weekly(id);


--
-- Name: trades trades_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.trades
    ADD CONSTRAINT trades_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

