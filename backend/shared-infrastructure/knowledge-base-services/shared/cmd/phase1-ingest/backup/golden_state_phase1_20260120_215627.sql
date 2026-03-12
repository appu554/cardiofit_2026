--
-- PostgreSQL database dump
--

\restrict N6TfcGjjyjKDqqii1xnd2rch6jpdbJTenFTxTSrTtf4SM8FcapKyOmKxv7rJZGO

-- Dumped from database version 15.15
-- Dumped by pg_dump version 15.15

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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: formulary_coverage; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.formulary_coverage (
    id integer NOT NULL,
    rxcui character varying(20) NOT NULL,
    drug_name character varying(500) NOT NULL,
    generic_name character varying(500),
    ndc character varying(20),
    contract_id character varying(20) NOT NULL,
    plan_id character varying(20) NOT NULL,
    segment_id character varying(20),
    plan_type character varying(50),
    on_formulary boolean DEFAULT true NOT NULL,
    tier integer,
    tier_level_code character varying(50),
    prior_auth boolean DEFAULT false,
    step_therapy boolean DEFAULT false,
    quantity_limit boolean DEFAULT false,
    quantity_limit_type character varying(50),
    quantity_limit_amt integer,
    quantity_limit_days integer,
    effective_date date,
    effective_year integer NOT NULL,
    fact_id uuid,
    source_version character varying(50),
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: formulary_coverage_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.formulary_coverage_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: formulary_coverage_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.formulary_coverage_id_seq OWNED BY public.formulary_coverage.id;


--
-- Name: ingestion_metadata; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.ingestion_metadata (
    id integer NOT NULL,
    source_name character varying(100) NOT NULL,
    source_version character varying(100) NOT NULL,
    records_loaded integer,
    records_skipped integer,
    records_failed integer,
    load_timestamp timestamp with time zone DEFAULT now(),
    sha256_checksum character varying(64),
    source_url text,
    load_duration_ms integer,
    notes text
);


--
-- Name: ingestion_metadata_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.ingestion_metadata_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: ingestion_metadata_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.ingestion_metadata_id_seq OWNED BY public.ingestion_metadata.id;


--
-- Name: interaction_matrix; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.interaction_matrix (
    id integer NOT NULL,
    drug1_rxcui character varying(20) NOT NULL,
    drug1_name character varying(500) NOT NULL,
    drug2_rxcui character varying(20) NOT NULL,
    drug2_name character varying(500) NOT NULL,
    severity character varying(50) NOT NULL,
    clinical_effect text,
    management text,
    mechanism character varying(255),
    documentation character varying(50),
    is_bidirectional boolean DEFAULT true,
    precipitant_rxcui character varying(20),
    object_rxcui character varying(20),
    interaction_mechanism character varying(255),
    source_dataset character varying(50) NOT NULL,
    source_pair_id character varying(50),
    evidence_level character varying(20),
    clinical_source character varying(100),
    fact_id uuid,
    source_version character varying(50),
    last_updated timestamp with time zone DEFAULT now(),
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: interaction_matrix_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.interaction_matrix_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: interaction_matrix_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.interaction_matrix_id_seq OWNED BY public.interaction_matrix.id;


--
-- Name: lab_reference_ranges; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.lab_reference_ranges (
    id integer NOT NULL,
    loinc_code character varying(20) NOT NULL,
    component character varying(255) NOT NULL,
    property character varying(50),
    time_aspect character varying(20),
    system character varying(100),
    scale_type character varying(20),
    method_type character varying(100),
    class character varying(100),
    short_name character varying(100),
    long_name text,
    unit character varying(50),
    low_normal numeric(10,4),
    high_normal numeric(10,4),
    critical_low numeric(10,4),
    critical_high numeric(10,4),
    age_group character varying(50),
    sex character varying(20),
    clinical_category character varying(50),
    interpretation_guidance text,
    delta_check_percent numeric(5,2),
    delta_check_hours integer,
    deprecated boolean DEFAULT false,
    fact_id uuid,
    source_version character varying(50),
    created_at timestamp with time zone DEFAULT now()
);


--
-- Name: lab_reference_ranges_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.lab_reference_ranges_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: lab_reference_ranges_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.lab_reference_ranges_id_seq OWNED BY public.lab_reference_ranges.id;


--
-- Name: formulary_coverage id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.formulary_coverage ALTER COLUMN id SET DEFAULT nextval('public.formulary_coverage_id_seq'::regclass);


--
-- Name: ingestion_metadata id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ingestion_metadata ALTER COLUMN id SET DEFAULT nextval('public.ingestion_metadata_id_seq'::regclass);


--
-- Name: interaction_matrix id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interaction_matrix ALTER COLUMN id SET DEFAULT nextval('public.interaction_matrix_id_seq'::regclass);


--
-- Name: lab_reference_ranges id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_reference_ranges ALTER COLUMN id SET DEFAULT nextval('public.lab_reference_ranges_id_seq'::regclass);


--
-- Data for Name: formulary_coverage; Type: TABLE DATA; Schema: public; Owner: -
--

COPY public.formulary_coverage (id, rxcui, drug_name, generic_name, ndc, contract_id, plan_id, segment_id, plan_type, on_formulary, tier, tier_level_code, prior_auth, step_therapy, quantity_limit, quantity_limit_type, quantity_limit_amt, quantity_limit_days, effective_date, effective_year, fact_id, source_version, created_at) FROM stdin;
1	6916	Metformin 500mg Tablet	\N	00555076802	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
2	6916	Metformin 850mg Tablet	\N	00555076901	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
3	6916	Metformin 1000mg Tablet	\N	00555077001	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
4	83367	Atorvastatin 10mg Tablet	\N	00071015523	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
5	83367	Atorvastatin 20mg Tablet	\N	00071015540	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
6	83367	Atorvastatin 40mg Tablet	\N	00071015580	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
7	83367	Atorvastatin 80mg Tablet	\N	00071015623	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
8	36567	Simvastatin 10mg Tablet	\N	00006073531	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
9	36567	Simvastatin 20mg Tablet	\N	00006073551	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
10	36567	Simvastatin 40mg Tablet	\N	00006073573	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
11	36567	Simvastatin 80mg Tablet	\N	00006073693	H1234	001	\N	\N	t	2	2	f	f	t	\N	30	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
12	313782	Rosuvastatin 5mg Tablet	\N	00093505501	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
13	313782	Rosuvastatin 10mg Tablet	\N	00093505601	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
14	313782	Rosuvastatin 20mg Tablet	\N	00093505801	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
15	313782	Rosuvastatin 40mg Tablet	\N	00093506001	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
16	52175	Losartan 25mg Tablet	\N	00378015501	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
17	52175	Losartan 50mg Tablet	\N	00378016001	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
18	52175	Losartan 100mg Tablet	\N	00378016501	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
19	3827	Enalapril 2.5mg Tablet	\N	00185001401	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
20	3827	Enalapril 5mg Tablet	\N	00185001501	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
21	3827	Enalapril 10mg Tablet	\N	00185001701	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
22	3827	Enalapril 20mg Tablet	\N	00185001801	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
23	29046	Lisinopril 5mg Tablet	\N	00378006501	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
24	29046	Lisinopril 10mg Tablet	\N	00378007001	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
25	29046	Lisinopril 20mg Tablet	\N	00378007501	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
26	29046	Lisinopril 40mg Tablet	\N	00378008001	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
27	866924	Amlodipine 2.5mg Tablet	\N	00591521730	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
28	866924	Amlodipine 5mg Tablet	\N	00591521830	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
29	866924	Amlodipine 10mg Tablet	\N	00591521930	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
30	197381	Ibuprofen 200mg Tablet	\N	00904588260	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
31	197381	Ibuprofen 400mg Tablet	\N	00904588460	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
32	197381	Ibuprofen 600mg Tablet	\N	00904588660	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
33	197381	Ibuprofen 800mg Tablet	\N	00904588860	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
34	221202	Naproxen 250mg Tablet	\N	00904515060	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
35	221202	Naproxen 500mg Tablet	\N	00904515260	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
36	161	Acetaminophen 325mg Tablet	\N	00054414425	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
37	161	Acetaminophen 500mg Tablet	\N	00054414525	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
38	1191	Aspirin 81mg Tablet	\N	00904203260	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
39	1191	Aspirin 325mg Tablet	\N	00904203360	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
40	11289	Warfarin 1mg Tablet	\N	00555075402	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
41	11289	Warfarin 2mg Tablet	\N	00555075502	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
42	11289	Warfarin 2.5mg Tablet	\N	00555075602	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
43	11289	Warfarin 5mg Tablet	\N	00555075702	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
44	11289	Warfarin 7.5mg Tablet	\N	00555075802	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
45	11289	Warfarin 10mg Tablet	\N	00555075902	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
46	114979	Clopidogrel 75mg Tablet	\N	68180051902	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
47	40790	Omeprazole 20mg Capsule	\N	00378180293	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
48	40790	Omeprazole 40mg Capsule	\N	00378180493	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
49	283742	Pantoprazole 20mg Tablet	\N	00378809293	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
50	283742	Pantoprazole 40mg Tablet	\N	00378809493	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
51	4441	Fluoxetine 10mg Capsule	\N	00555089502	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
52	4441	Fluoxetine 20mg Capsule	\N	00555089602	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
53	4441	Fluoxetine 40mg Capsule	\N	00555089702	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
54	42347	Sertraline 25mg Tablet	\N	00378413601	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
55	42347	Sertraline 50mg Tablet	\N	00378413701	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
56	42347	Sertraline 100mg Tablet	\N	00378413801	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
57	32937	Paroxetine 10mg Tablet	\N	00591375610	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
58	32937	Paroxetine 20mg Tablet	\N	00591375710	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
59	32937	Paroxetine 40mg Tablet	\N	00591375910	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
60	1373	Carbamazepine 200mg Tablet	\N	00093086201	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
61	8123	Phenytoin 100mg Capsule	\N	00071037132	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
62	28728	Valproic Acid 250mg Capsule	\N	00074685213	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
63	28728	Valproic Acid 500mg Capsule	\N	00074685313	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
64	1813	Lamotrigine 25mg Tablet	\N	00173064255	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
65	1813	Lamotrigine 100mg Tablet	\N	00173064355	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
66	1813	Lamotrigine 150mg Tablet	\N	00173064455	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
67	1813	Lamotrigine 200mg Tablet	\N	00173064555	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
68	115698	Gabapentin 100mg Capsule	\N	00071080524	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
69	115698	Gabapentin 300mg Capsule	\N	00071080640	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
70	115698	Gabapentin 400mg Capsule	\N	00071080740	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
71	73178	Pregabalin 25mg Capsule	\N	00071101568	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
72	73178	Pregabalin 75mg Capsule	\N	00071101668	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
73	73178	Pregabalin 150mg Capsule	\N	00071101768	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
74	196503	Clarithromycin 250mg Tablet	\N	00074301913	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
75	196503	Clarithromycin 500mg Tablet	\N	00074302013	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
76	2348	Ciprofloxacin 250mg Tablet	\N	51079089020	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
77	2348	Ciprofloxacin 500mg Tablet	\N	51079089120	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
78	2348	Ciprofloxacin 750mg Tablet	\N	51079089220	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
79	139462	Levofloxacin 250mg Tablet	\N	00093311101	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
80	139462	Levofloxacin 500mg Tablet	\N	00093311201	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
81	139462	Levofloxacin 750mg Tablet	\N	00093311301	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
82	6813	Methotrexate 2.5mg Tablet	\N	00054413325	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
83	5640	Lithium Carbonate 300mg Capsule	\N	00054003525	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
84	5640	Lithium Carbonate 450mg Tablet ER	\N	00054003625	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
85	2551	Digoxin 0.125mg Tablet	\N	00173028200	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
86	2551	Digoxin 0.25mg Tablet	\N	00173028300	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
87	17767	Amiodarone 200mg Tablet	\N	00008424101	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
88	7052	Morphine Sulfate 15mg Tablet IR	\N	00054414029	H1234	001	\N	\N	t	2	2	t	f	t	\N	120	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
89	7052	Morphine Sulfate 30mg Tablet IR	\N	00054414129	H1234	001	\N	\N	t	2	2	t	f	t	\N	120	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
90	7804	Oxycodone 5mg Tablet IR	\N	00054416425	H1234	001	\N	\N	t	2	2	t	f	t	\N	120	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
91	7804	Oxycodone 10mg Tablet IR	\N	00054416525	H1234	001	\N	\N	t	2	2	t	f	t	\N	90	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
92	7804	Oxycodone 15mg Tablet IR	\N	00054416625	H1234	001	\N	\N	t	2	2	t	f	t	\N	90	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
93	7804	Oxycodone 20mg Tablet IR	\N	00054416725	H1234	001	\N	\N	t	2	2	t	f	t	\N	60	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
94	5489	Hydrocodone/APAP 5/325mg Tablet	\N	00591052110	H1234	001	\N	\N	t	2	2	f	f	t	\N	120	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
95	5489	Hydrocodone/APAP 7.5/325mg Tablet	\N	00591052210	H1234	001	\N	\N	t	2	2	f	f	t	\N	90	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
96	5489	Hydrocodone/APAP 10/325mg Tablet	\N	00591052310	H1234	001	\N	\N	t	2	2	t	f	t	\N	60	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
97	10689	Tramadol 50mg Tablet	\N	00591052501	H1234	001	\N	\N	t	2	2	f	f	t	\N	120	30	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
98	4337	Fluconazole 50mg Tablet	\N	00049344028	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
99	4337	Fluconazole 100mg Tablet	\N	00049344128	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
100	4337	Fluconazole 150mg Tablet	\N	00049344228	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
101	4337	Fluconazole 200mg Tablet	\N	00049344328	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
102	203150	Tacrolimus 0.5mg Capsule	\N	00469061711	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
103	203150	Tacrolimus 1mg Capsule	\N	00469061611	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
104	203150	Tacrolimus 5mg Capsule	\N	00469061911	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
105	3640	Cyclosporine 25mg Capsule	\N	00078024915	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
106	3640	Cyclosporine 100mg Capsule	\N	00078024815	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
107	8356	Theophylline 100mg Tablet ER	\N	00085013301	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
108	8356	Theophylline 200mg Tablet ER	\N	00085013401	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
109	8356	Theophylline 300mg Tablet ER	\N	00085013501	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
110	8331	Potassium Chloride 10mEq Tablet ER	\N	00591061210	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
111	8331	Potassium Chloride 20mEq Tablet ER	\N	00591061310	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
112	29046	Spironolactone 25mg Tablet	\N	00591200101	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
113	29046	Spironolactone 50mg Tablet	\N	00591200201	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
114	29046	Spironolactone 100mg Tablet	\N	00591200401	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
115	866514	Metoprolol Succinate 25mg Tablet ER	\N	00378402077	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
116	866514	Metoprolol Succinate 50mg Tablet ER	\N	00378402177	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
117	866514	Metoprolol Succinate 100mg Tablet ER	\N	00378402377	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
118	866514	Metoprolol Succinate 200mg Tablet ER	\N	00378402477	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
119	33738	Metoprolol Tartrate 25mg Tablet	\N	00591037710	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
120	33738	Metoprolol Tartrate 50mg Tablet	\N	00591037810	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
121	33738	Metoprolol Tartrate 100mg Tablet	\N	00591037910	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
122	149	Atenolol 25mg Tablet	\N	00185041001	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
123	149	Atenolol 50mg Tablet	\N	00185041101	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
124	149	Atenolol 100mg Tablet	\N	00185041201	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
125	20610	Carvedilol 3.125mg Tablet	\N	00378027301	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
126	20610	Carvedilol 6.25mg Tablet	\N	00378027401	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
127	20610	Carvedilol 12.5mg Tablet	\N	00378027501	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
128	20610	Carvedilol 25mg Tablet	\N	00378027601	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
129	1098	Furosemide 20mg Tablet	\N	00378252301	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
130	1098	Furosemide 40mg Tablet	\N	00378252401	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
131	1098	Furosemide 80mg Tablet	\N	00378252501	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
132	5487	Hydrochlorothiazide 12.5mg Capsule	\N	00378117101	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
133	5487	Hydrochlorothiazide 25mg Tablet	\N	00378117201	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
134	5487	Hydrochlorothiazide 50mg Tablet	\N	00378117301	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
135	860975	Levothyroxine 25mcg Tablet	\N	00591552101	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
136	860975	Levothyroxine 50mcg Tablet	\N	00591552201	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
137	860975	Levothyroxine 75mcg Tablet	\N	00591552301	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
138	860975	Levothyroxine 100mcg Tablet	\N	00591552401	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
139	860975	Levothyroxine 125mcg Tablet	\N	00591552501	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
140	860975	Levothyroxine 150mcg Tablet	\N	00591552601	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
141	4815	Glipizide 5mg Tablet	\N	00093314601	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
142	4815	Glipizide 10mg Tablet	\N	00093314701	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
143	4821	Glyburide 1.25mg Tablet	\N	00591501801	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
144	4821	Glyburide 2.5mg Tablet	\N	00591501901	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
145	4821	Glyburide 5mg Tablet	\N	00591502001	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
146	25789	Glimepiride 1mg Tablet	\N	00093701706	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
147	25789	Glimepiride 2mg Tablet	\N	00093701806	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
148	25789	Glimepiride 4mg Tablet	\N	00093701906	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
149	593411	Insulin Glargine (Lantus) 100U/mL	\N	00002147180	H1234	001	\N	\N	t	3	3	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
150	1368001	Insulin Lispro (Humalog) 100U/mL	\N	00169444112	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
151	5856	Regular Insulin (Humulin R) 100U/mL	\N	00002751001	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
152	5856	NPH Insulin (Humulin N) 100U/mL	\N	00002821501	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
153	1361029	Insulin Degludec (Tresiba) 100U/mL	\N	00169750111	H1234	001	\N	\N	t	4	4	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
154	1373458	Empagliflozin (Jardiance) 10mg	\N	00597019701	H1234	001	\N	\N	t	4	4	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
155	1373458	Empagliflozin (Jardiance) 25mg	\N	00597019801	H1234	001	\N	\N	t	4	4	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
156	1545653	Dapagliflozin (Farxiga) 5mg	\N	00078066115	H1234	001	\N	\N	t	4	4	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
157	1545653	Dapagliflozin (Farxiga) 10mg	\N	00078066215	H1234	001	\N	\N	t	4	4	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
158	1664314	Semaglutide (Ozempic) 0.5mg Pen	\N	00169412312	H1234	001	\N	\N	t	5	5	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
159	1664314	Semaglutide (Ozempic) 1mg Pen	\N	00169412412	H1234	001	\N	\N	t	5	5	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
160	897122	Liraglutide (Victoza) 18mg/3mL Pen	\N	00002143780	H1234	001	\N	\N	t	4	4	t	t	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
161	1992	Albuterol Inhaler 90mcg	\N	00186077660	H1234	001	\N	\N	t	1	1	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
162	745679	Fluticasone/Salmeterol (Advair) 250/50	\N	00085462303	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
163	896188	Tiotropium (Spiriva) 18mcg Capsule	\N	00597012360	H1234	001	\N	\N	t	3	3	t	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
164	324089	Montelukast 10mg Tablet	\N	00074622710	H1234	001	\N	\N	t	2	2	f	f	f	\N	\N	\N	\N	2024	\N	CMS-2024	2026-01-20 16:26:13.113206+00
\.


--
-- Data for Name: ingestion_metadata; Type: TABLE DATA; Schema: public; Owner: -
--

COPY public.ingestion_metadata (id, source_name, source_version, records_loaded, records_skipped, records_failed, load_timestamp, sha256_checksum, source_url, load_duration_ms, notes) FROM stdin;
1	KB-5-ONC-DDI	ONC-2024-Q4	176	0	0	2026-01-20 16:04:00.75147+00	\N	\N	\N	Phase 1 ETL: ONC High-Priority Drug-Drug Interactions (bidirectional pairs)
2	KB-6-CMS-Formulary	CMS-2024	164	0	0	2026-01-20 16:04:00.75147+00	\N	\N	\N	Phase 1 ETL: CMS Medicare Part D Formulary (NOT_COVERED filtered per governance)
3	KB-16-LOINC-Labs	LOINC-2024	50	0	0	2026-01-20 16:04:00.75147+00	\N	\N	\N	Phase 1 ETL: LOINC Lab Reference Ranges (deprecated codes skipped)
4	KB-5-ONC-DDI	ONC-2024-Q4	176	0	0	2026-01-20 16:26:13.113206+00	\N	\N	\N	Phase 1 ETL: ONC High-Priority Drug-Drug Interactions (bidirectional pairs)
5	KB-6-CMS-Formulary	CMS-2024	164	0	0	2026-01-20 16:26:13.113206+00	\N	\N	\N	Phase 1 ETL: CMS Medicare Part D Formulary (NOT_COVERED filtered per governance)
6	KB-16-LOINC-Labs	LOINC-2024	50	0	0	2026-01-20 16:26:13.113206+00	\N	\N	\N	Phase 1 ETL: LOINC Lab Reference Ranges (deprecated codes skipped)
\.


--
-- Data for Name: interaction_matrix; Type: TABLE DATA; Schema: public; Owner: -
--

COPY public.interaction_matrix (id, drug1_rxcui, drug1_name, drug2_rxcui, drug2_name, severity, clinical_effect, management, mechanism, documentation, is_bidirectional, precipitant_rxcui, object_rxcui, interaction_mechanism, source_dataset, source_pair_id, evidence_level, clinical_source, fact_id, source_version, last_updated, created_at) FROM stdin;
2	11289	Warfarin	261106	Aspirin	CONTRAINDICATED	Increased risk of bleeding due to antiplatelet and anticoagulant effects	Avoid combination unless benefits outweigh risks. Monitor for signs of bleeding.	\N	Established interaction based on mechanism and cli	t	11289	261106	\N	ONC_HIGH_PRIORITY	ONC-001	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
3	11289	Warfarin	197381	Ibuprofen	HIGH	NSAIDs increase anticoagulant effect and bleeding risk	Avoid NSAIDs with warfarin. If unavoidable monitor INR closely and watch for bleeding.	\N	Well-documented interaction with clinical signific	t	11289	197381	\N	ONC_HIGH_PRIORITY	ONC-002	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
5	11289	Warfarin	221202	Naproxen	HIGH	NSAIDs increase anticoagulant effect and GI bleeding risk	Avoid combination. If necessary use lowest NSAID dose for shortest duration.	\N	Class effect with NSAIDs	t	11289	221202	\N	ONC_HIGH_PRIORITY	ONC-004	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
7	11289	Warfarin	9068	Sulfamethoxazole/TMP	HIGH	Multiple mechanisms increase warfarin effect	Monitor INR closely when starting or stopping TMP/SMX.	\N	Clinical trial data supports interaction	t	11289	9068	\N	ONC_HIGH_PRIORITY	ONC-006	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
26	5640	Lithium	52175	Losartan	HIGH	ARBs may reduce lithium clearance similar to ACE inhibitors	Monitor lithium levels when starting ARBs.	\N	Class effect expected	t	5640	52175	\N	ONC_HIGH_PRIORITY	ONC-025	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
27	36567	Simvastatin	196503	Clarithromycin	CONTRAINDICATED	Clarithromycin is a strong CYP3A4 inhibitor increasing statin myopathy risk	Avoid combination. Use azithromycin as alternative or suspend statin therapy.	\N	FDA boxed warning for myopathy risk	t	36567	196503	\N	ONC_HIGH_PRIORITY	ONC-026	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
28	36567	Simvastatin	28439	Erythromycin	CONTRAINDICATED	Erythromycin inhibits CYP3A4 increasing statin exposure and myopathy risk	Avoid combination. Consider alternative antibiotic or statin.	\N	Well-documented CYP3A4 interaction	t	36567	28439	\N	ONC_HIGH_PRIORITY	ONC-027	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
31	36567	Simvastatin	203150	Cyclosporine	CONTRAINDICATED	Cyclosporine dramatically increases statin levels	Avoid combination. Very high myopathy/rhabdomyolysis risk.	\N	FDA contraindication	t	36567	203150	\N	ONC_HIGH_PRIORITY	ONC-030	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
52	8331	Potassium Chloride	3827	Enalapril	HIGH	ACE inhibitors reduce potassium excretion	Monitor potassium when using supplements with ACE inhibitors.	\N	Documented additive effect	t	8331	3827	\N	ONC_HIGH_PRIORITY	ONC-051	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
53	8331	Potassium Chloride	52175	Losartan	HIGH	ARBs reduce potassium excretion similar to ACE inhibitors	Monitor potassium with potassium supplements.	\N	Class effect	t	8331	52175	\N	ONC_HIGH_PRIORITY	ONC-052	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
54	29046	Spironolactone	3827	Enalapril	HIGH	Additive hyperkalemia risk from dual RAAS blockade	Monitor potassium and renal function closely. Avoid in high-risk patients.	\N	ONTARGET trial data	t	29046	3827	\N	ONC_HIGH_PRIORITY	ONC-053	HIGH	Clinical Trial	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
55	29046	Spironolactone	52175	Losartan	HIGH	Additive hyperkalemia risk from dual RAAS blockade	Monitor potassium closely. Generally avoid in CKD.	\N	Class effect	t	29046	52175	\N	ONC_HIGH_PRIORITY	ONC-054	HIGH	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
57	7052	Morphine	1819	Benzodiazepines	HIGH	Additive CNS and respiratory depression risk of death	Avoid combination. If necessary use lowest effective doses and monitor.	\N	FDA boxed warning	t	7052	1819	\N	ONC_HIGH_PRIORITY	ONC-056	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
85	17767	Amiodarone	4441	Fluoxetine	HIGH	QT prolongation risk from both drugs	Monitor QT interval. Avoid if possible.	\N	Additive QT effect	t	17767	4441	\N	ONC_HIGH_PRIORITY	ONC-084	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
88	35636	Thioridazine	4337	Fluconazole	HIGH	QT prolongation from both drugs	Avoid combination due to arrhythmia risk.	\N	Additive QT effect	t	35636	4337	\N	ONC_HIGH_PRIORITY	ONC-087	HIGH	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
89	8787	Pimozide	196503	Clarithromycin	CONTRAINDICATED	Severe QT prolongation and cardiac arrest risk	Absolutely contraindicated.	\N	FDA contraindication	t	8787	196503	\N	ONC_HIGH_PRIORITY	ONC-088	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
90	8787	Pimozide	28439	Erythromycin	CONTRAINDICATED	QT prolongation risk from CYP3A4 inhibition	Absolutely contraindicated.	\N	FDA contraindication	t	8787	28439	\N	ONC_HIGH_PRIORITY	ONC-089	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
91	8787	Pimozide	4337	Fluconazole	CONTRAINDICATED	QT prolongation from CYP3A4 inhibition	Contraindicated. Use alternative antifungal.	\N	FDA warning	t	8787	4337	\N	ONC_HIGH_PRIORITY	ONC-090	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
92	196503	Clarithromycin	48937	Colchicine	CONTRAINDICATED	Clarithromycin dramatically increases colchicine levels	Contraindicated especially with renal impairment. Fatal cases reported.	\N	FDA warning	t	196503	48937	\N	ONC_HIGH_PRIORITY	ONC-091	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
93	196503	Clarithromycin	228029	Dronedarone	CONTRAINDICATED	CYP3A4 inhibition increases dronedarone levels and QT prolongation	Contraindicated combination.	\N	FDA contraindication	t	196503	228029	\N	ONC_HIGH_PRIORITY	ONC-092	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
8	11289	Warfarin	6851	Metronidazole	HIGH	Metronidazole inhibits warfarin metabolism	Monitor INR closely. May need warfarin dose reduction.	\N	Well-documented interaction	t	11289	6851	\N	ONC_HIGH_PRIORITY	ONC-007	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
10	197381	Ibuprofen	1191	Aspirin	MODERATE	Combined NSAID use increases GI bleeding risk	Avoid combining NSAIDs when possible. Use gastroprotection if necessary.	\N	Well-established additive toxicity	t	197381	1191	\N	ONC_HIGH_PRIORITY	ONC-009	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
11	197381	Ibuprofen	6916	Metformin	MODERATE	NSAIDs may reduce metformin efficacy and increase lactic acidosis risk	Monitor renal function and blood glucose.	\N	Theoretical concern with limited evidence	t	197381	6916	\N	ONC_HIGH_PRIORITY	ONC-010	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
12	197381	Ibuprofen	5640	Lithium	HIGH	NSAIDs decrease lithium clearance by 15-20%	Monitor lithium levels when starting or stopping NSAIDs. Adjust dose as needed.	\N	Documented pharmacokinetic interaction	t	197381	5640	\N	ONC_HIGH_PRIORITY	ONC-011	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
13	6813	Methotrexate	197381	Ibuprofen	HIGH	NSAIDs decrease methotrexate clearance increasing toxicity risk	Avoid NSAIDs during high-dose methotrexate. Monitor for methotrexate toxicity.	\N	Documented pharmacokinetic interaction	t	6813	197381	\N	ONC_HIGH_PRIORITY	ONC-012	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
14	6813	Methotrexate	9524	Trimethoprim	HIGH	Both drugs inhibit folate metabolism leading to severe bone marrow suppression	Avoid combination or use with extreme caution. Monitor CBC frequently.	\N	Additive antifolate toxicity	t	6813	9524	\N	ONC_HIGH_PRIORITY	ONC-013	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
15	6813	Methotrexate	8123	Phenytoin	HIGH	Complex interaction affecting methotrexate and phenytoin levels	Monitor both drug levels and adjust doses as needed.	\N	Documented bidirectional interaction	t	6813	8123	\N	ONC_HIGH_PRIORITY	ONC-014	MODERATE	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
16	6813	Methotrexate	38413	Probenecid	HIGH	Probenecid decreases methotrexate renal clearance	Avoid combination with high-dose methotrexate. Monitor for toxicity.	\N	Pharmacokinetic interaction	t	6813	38413	\N	ONC_HIGH_PRIORITY	ONC-015	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
18	2551	Digoxin	10600	Verapamil	HIGH	Verapamil increases digoxin levels and additive bradycardia	Reduce digoxin dose by 25-50%. Monitor heart rate and digoxin levels.	\N	Documented interaction with clinical significance	t	2551	10600	\N	ONC_HIGH_PRIORITY	ONC-017	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
19	2551	Digoxin	17128	Diltiazem	HIGH	Diltiazem increases digoxin levels and additive AV nodal effects	Monitor digoxin levels and heart rate. May need dose adjustment.	\N	Similar mechanism to verapamil	t	2551	17128	\N	ONC_HIGH_PRIORITY	ONC-018	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
20	2551	Digoxin	6754	Quinidine	CONTRAINDICATED	Quinidine increases digoxin levels by 50-100% via multiple mechanisms	Reduce digoxin dose by 50%. Monitor digoxin levels and QT interval.	\N	Well-established dangerous interaction	t	2551	6754	\N	ONC_HIGH_PRIORITY	ONC-019	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
21	2551	Digoxin	29046	Spironolactone	MODERATE	Spironolactone may interfere with digoxin assay and renal clearance	Monitor digoxin levels. Be aware of assay interference.	\N	Mixed evidence	t	2551	29046	\N	ONC_HIGH_PRIORITY	ONC-020	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
22	6916	Metformin	20352	Contrast Media (Iodinated)	HIGH	Metformin accumulation may cause lactic acidosis with renal impairment from contrast	Hold metformin before contrast procedure. Resume 48h after if renal function stable.	\N	Guidelines-based recommendation	t	6916	20352	\N	ONC_HIGH_PRIORITY	ONC-021	HIGH	ACR Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
23	6916	Metformin	82122	Alcohol	MODERATE	Alcohol increases lactic acidosis risk with metformin	Avoid excessive alcohol consumption with metformin.	\N	Mechanism-based concern	t	6916	82122	\N	ONC_HIGH_PRIORITY	ONC-022	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
24	5640	Lithium	1819	Diuretics (Thiazide)	HIGH	Thiazides reduce lithium clearance causing toxicity	Monitor lithium levels closely. May need to reduce lithium dose.	\N	Well-documented interaction	t	5640	1819	\N	ONC_HIGH_PRIORITY	ONC-023	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
25	5640	Lithium	3827	Enalapril	HIGH	ACE inhibitors reduce lithium clearance	Monitor lithium levels when starting ACE inhibitors.	\N	Class effect with ACE inhibitors	t	5640	3827	\N	ONC_HIGH_PRIORITY	ONC-024	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
32	36567	Simvastatin	261962	Grapefruit Juice	HIGH	Grapefruit inhibits intestinal CYP3A4 increasing statin absorption	Avoid grapefruit juice with simvastatin.	\N	Food-drug interaction	t	36567	261962	\N	ONC_HIGH_PRIORITY	ONC-031	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
33	83367	Atorvastatin	196503	Clarithromycin	HIGH	CYP3A4 inhibition increases atorvastatin exposure and myopathy risk	Limit atorvastatin to 20mg daily with clarithromycin. Monitor for muscle symptoms.	\N	FDA labeling restriction	t	83367	196503	\N	ONC_HIGH_PRIORITY	ONC-032	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
34	83367	Atorvastatin	3640	Cyclosporine	CONTRAINDICATED	Cyclosporine significantly increases atorvastatin levels	Limit atorvastatin to 10mg daily with cyclosporine if used.	\N	FDA label restriction	t	83367	3640	\N	ONC_HIGH_PRIORITY	ONC-033	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
35	4441	Fluoxetine	6470	MAO Inhibitors	CONTRAINDICATED	Serotonin syndrome - potentially fatal	Contraindicated. Allow 5 weeks washout between fluoxetine and MAOIs.	\N	Life-threatening interaction	t	4441	6470	\N	ONC_HIGH_PRIORITY	ONC-034	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
37	4441	Fluoxetine	8787	Pimozide	CONTRAINDICATED	Fluoxetine inhibits pimozide metabolism increasing QT prolongation risk	Contraindicated combination. Risk of fatal arrhythmia.	\N	FDA contraindication	t	4441	8787	\N	ONC_HIGH_PRIORITY	ONC-036	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
39	42347	Sertraline	6470	MAO Inhibitors	CONTRAINDICATED	Serotonin syndrome risk	Contraindicated. Allow 2 weeks washout between sertraline and MAOIs.	\N	Life-threatening interaction	t	42347	6470	\N	ONC_HIGH_PRIORITY	ONC-038	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
40	42347	Sertraline	8787	Pimozide	CONTRAINDICATED	Sertraline inhibits pimozide metabolism causing QT prolongation	Contraindicated. Use alternative antipsychotic.	\N	FDA contraindication	t	42347	8787	\N	ONC_HIGH_PRIORITY	ONC-039	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
41	32937	Paroxetine	6470	MAO Inhibitors	CONTRAINDICATED	Serotonin syndrome risk	Contraindicated. Allow 2 weeks washout.	\N	Life-threatening interaction	t	32937	6470	\N	ONC_HIGH_PRIORITY	ONC-040	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
42	32937	Paroxetine	8787	Pimozide	CONTRAINDICATED	Paroxetine inhibits pimozide metabolism	Contraindicated. Risk of fatal arrhythmia.	\N	FDA contraindication	t	32937	8787	\N	ONC_HIGH_PRIORITY	ONC-041	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
44	114979	Clopidogrel	40790	Omeprazole	HIGH	Omeprazole reduces clopidogrel activation via CYP2C19 inhibition	Use pantoprazole instead of omeprazole. Monitor for cardiovascular events.	\N	FDA warning on reduced antiplatelet effect	t	114979	40790	\N	ONC_HIGH_PRIORITY	ONC-043	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
45	114979	Clopidogrel	28439	Esomeprazole	HIGH	Esomeprazole reduces clopidogrel activation similar to omeprazole	Prefer pantoprazole or other alternatives.	\N	Class effect with some PPIs	t	114979	28439	\N	ONC_HIGH_PRIORITY	ONC-044	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
46	114979	Clopidogrel	4337	Fluconazole	HIGH	CYP2C19 inhibition reduces clopidogrel active metabolite formation	Avoid combination if possible. Consider alternative antifungal.	\N	Pharmacokinetic interaction	t	114979	4337	\N	ONC_HIGH_PRIORITY	ONC-045	MODERATE	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
47	3640	Cyclosporine	196503	Clarithromycin	HIGH	CYP3A4 inhibition significantly increases cyclosporine levels	Monitor cyclosporine levels closely. May need 50% dose reduction.	\N	Well-documented nephrotoxicity risk	t	3640	196503	\N	ONC_HIGH_PRIORITY	ONC-046	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
48	3640	Cyclosporine	1373	Carbamazepine	HIGH	Carbamazepine induces CYP3A4 decreasing cyclosporine levels	Monitor cyclosporine levels. May need dose increase.	\N	CYP3A4 induction	t	3640	1373	\N	ONC_HIGH_PRIORITY	ONC-047	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
49	3640	Cyclosporine	8123	Phenytoin	HIGH	Phenytoin induces cyclosporine metabolism	Monitor cyclosporine levels closely during phenytoin therapy.	\N	CYP3A4 induction	t	3640	8123	\N	ONC_HIGH_PRIORITY	ONC-048	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
50	3640	Cyclosporine	8331	Potassium Chloride	HIGH	Cyclosporine causes potassium retention	Monitor potassium levels. Avoid potassium supplements if possible.	\N	Additive hyperkalemia risk	t	3640	8331	\N	ONC_HIGH_PRIORITY	ONC-049	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
51	8331	Potassium Chloride	29046	Spironolactone	HIGH	Additive hyperkalemia risk	Avoid combination or monitor potassium closely. Use lowest effective doses.	\N	Additive effect on potassium	t	8331	29046	\N	ONC_HIGH_PRIORITY	ONC-050	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
56	6754	Meperidine	6470	MAO Inhibitors	CONTRAINDICATED	Severe potentially fatal reactions including serotonin syndrome and hypertensive crisis	Absolutely contraindicated. Use alternative analgesics.	\N	Life-threatening interaction	t	6754	6470	\N	ONC_HIGH_PRIORITY	ONC-055	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
58	7052	Morphine	115698	Gabapentin	HIGH	Additive CNS depression and respiratory depression	Use lowest doses. Monitor respiratory status.	\N	Emerging evidence of risk	t	7052	115698	\N	ONC_HIGH_PRIORITY	ONC-057	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
59	7052	Morphine	73178	Pregabalin	HIGH	Additive CNS and respiratory depression	Use lowest doses. Monitor closely.	\N	Class effect with gabapentinoids	t	7052	73178	\N	ONC_HIGH_PRIORITY	ONC-058	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
60	5489	Hydrocodone	1819	Benzodiazepines	HIGH	Additive CNS and respiratory depression	Avoid combination. FDA boxed warning.	\N	FDA boxed warning	t	5489	1819	\N	ONC_HIGH_PRIORITY	ONC-059	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
61	5489	Hydrocodone	115698	Gabapentin	HIGH	Additive CNS depression	Use lowest effective doses and monitor.	\N	Gabapentinoid + opioid risk	t	5489	115698	\N	ONC_HIGH_PRIORITY	ONC-060	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
62	7804	Oxycodone	1819	Benzodiazepines	HIGH	Additive CNS and respiratory depression risk of death	Avoid combination. FDA boxed warning on concomitant use.	\N	FDA boxed warning	t	7804	1819	\N	ONC_HIGH_PRIORITY	ONC-061	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
63	7804	Oxycodone	115698	Gabapentin	HIGH	Additive CNS depression	Use lowest doses and monitor.	\N	Gabapentinoid + opioid risk	t	7804	115698	\N	ONC_HIGH_PRIORITY	ONC-062	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
64	3289	Codeine	1819	Benzodiazepines	HIGH	Additive CNS and respiratory depression	Avoid combination or use lowest doses.	\N	FDA boxed warning applies	t	3289	1819	\N	ONC_HIGH_PRIORITY	ONC-063	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
65	10689	Tramadol	6470	MAO Inhibitors	CONTRAINDICATED	Serotonin syndrome risk	Contraindicated. Allow washout period.	\N	Life-threatening interaction	t	10689	6470	\N	ONC_HIGH_PRIORITY	ONC-064	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
66	10689	Tramadol	1819	Benzodiazepines	HIGH	Additive CNS and respiratory depression	Avoid combination. Use lowest doses if necessary.	\N	FDA warning	t	10689	1819	\N	ONC_HIGH_PRIORITY	ONC-065	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
68	8356	Theophylline	196503	Clarithromycin	HIGH	Clarithromycin inhibits theophylline metabolism	Monitor theophylline levels. May need dose reduction.	\N	CYP3A4 interaction	t	8356	196503	\N	ONC_HIGH_PRIORITY	ONC-067	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
69	8356	Theophylline	28439	Erythromycin	HIGH	Erythromycin inhibits theophylline metabolism	Monitor theophylline levels closely.	\N	Well-documented interaction	t	8356	28439	\N	ONC_HIGH_PRIORITY	ONC-068	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
70	8356	Theophylline	2348	Ciprofloxacin	HIGH	Ciprofloxacin inhibits theophylline metabolism significantly	Reduce theophylline dose by 30-50% or avoid combination.	\N	CYP1A2 inhibition	t	8356	2348	\N	ONC_HIGH_PRIORITY	ONC-069	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
71	8356	Theophylline	1373	Carbamazepine	HIGH	Carbamazepine induces theophylline metabolism	Monitor theophylline levels. May need dose increase.	\N	CYP induction	t	8356	1373	\N	ONC_HIGH_PRIORITY	ONC-070	MODERATE	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
72	1373	Carbamazepine	11289	Warfarin	HIGH	Carbamazepine induces warfarin metabolism	Monitor INR closely. May need warfarin dose increase.	\N	CYP induction	t	1373	11289	\N	ONC_HIGH_PRIORITY	ONC-071	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
74	1373	Carbamazepine	196503	Clarithromycin	HIGH	Clarithromycin increases carbamazepine levels causing toxicity	Avoid combination or reduce carbamazepine dose.	\N	CYP3A4 inhibition	t	1373	196503	\N	ONC_HIGH_PRIORITY	ONC-073	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
75	1373	Carbamazepine	28439	Erythromycin	HIGH	Erythromycin increases carbamazepine levels	Avoid combination or monitor levels closely.	\N	CYP3A4 inhibition	t	1373	28439	\N	ONC_HIGH_PRIORITY	ONC-074	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
77	28728	Valproic Acid	1813	Lamotrigine	HIGH	Valproic acid doubles lamotrigine levels	Start lamotrigine at 25mg every other day. Slow titration required.	\N	FDA labeling restriction	t	28728	1813	\N	ONC_HIGH_PRIORITY	ONC-076	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
94	196503	Clarithromycin	321988	Ranolazine	CONTRAINDICATED	CYP3A4 inhibition significantly increases ranolazine levels	Contraindicated. Risk of QT prolongation.	\N	FDA contraindication	t	196503	321988	\N	ONC_HIGH_PRIORITY	ONC-093	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
95	28439	Erythromycin	48937	Colchicine	HIGH	Erythromycin increases colchicine levels	Reduce colchicine dose. Avoid in renal/hepatic impairment.	\N	CYP3A4 interaction	t	28439	48937	\N	ONC_HIGH_PRIORITY	ONC-094	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
96	28439	Erythromycin	321988	Ranolazine	HIGH	Erythromycin increases ranolazine levels	Limit ranolazine dose. Monitor QT interval.	\N	CYP3A4 interaction	t	28439	321988	\N	ONC_HIGH_PRIORITY	ONC-095	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
97	4337	Fluconazole	196503	Cisapride	CONTRAINDICATED	QT prolongation and torsades de pointes risk	Contraindicated. Cisapride restricted availability.	\N	FDA withdrawal related	t	4337	196503	\N	ONC_HIGH_PRIORITY	ONC-096	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
101	203150	Tacrolimus	196503	Clarithromycin	HIGH	CYP3A4 inhibition dramatically increases tacrolimus levels	Monitor tacrolimus levels closely. Significant dose reduction needed.	\N	Nephrotoxicity risk	t	203150	196503	\N	ONC_HIGH_PRIORITY	ONC-100	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
102	261106	Aspirin	11289	Warfarin	CONTRAINDICATED	Increased risk of bleeding due to antiplatelet and anticoagulant effects	Avoid combination unless benefits outweigh risks. Monitor for signs of bleeding.	\N	Established interaction based on mechanism and cli	t	261106	11289	\N	ONC_HIGH_PRIORITY	ONC-001-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
103	197381	Ibuprofen	11289	Warfarin	HIGH	NSAIDs increase anticoagulant effect and bleeding risk	Avoid NSAIDs with warfarin. If unavoidable monitor INR closely and watch for bleeding.	\N	Well-documented interaction with clinical signific	t	197381	11289	\N	ONC_HIGH_PRIORITY	ONC-002-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
79	8123	Phenytoin	11289	Warfarin	HIGH	Complex bidirectional interaction affecting metabolism of both drugs	Monitor INR and phenytoin levels closely. Adjust doses as needed.	\N	Well-documented	t	8123	11289	\N	ONC_HIGH_PRIORITY	ONC-078	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
105	221202	Naproxen	11289	Warfarin	HIGH	NSAIDs increase anticoagulant effect and GI bleeding risk	Avoid combination. If necessary use lowest NSAID dose for shortest duration.	\N	Class effect with NSAIDs	t	221202	11289	\N	ONC_HIGH_PRIORITY	ONC-004-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
98	4337	Fluconazole	11289	Warfarin	HIGH	Fluconazole inhibits CYP2C9 increasing warfarin levels	Reduce warfarin dose by 25-50%. Monitor INR closely.	\N	Pharmacokinetic interaction	t	4337	11289	\N	ONC_HIGH_PRIORITY	ONC-097	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
107	9068	Sulfamethoxazole/TMP	11289	Warfarin	HIGH	Multiple mechanisms increase warfarin effect	Monitor INR closely when starting or stopping TMP/SMX.	\N	Clinical trial data supports interaction	t	9068	11289	\N	ONC_HIGH_PRIORITY	ONC-006-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
108	6851	Metronidazole	11289	Warfarin	HIGH	Metronidazole inhibits warfarin metabolism	Monitor INR closely. May need warfarin dose reduction.	\N	Well-documented interaction	t	6851	11289	\N	ONC_HIGH_PRIORITY	ONC-007-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
84	17767	Amiodarone	11289	Warfarin	CONTRAINDICATED	Amiodarone inhibits CYP2C9 and CYP3A4 significantly increasing warfarin effect	Reduce warfarin dose by 30-50%. Interaction persists for weeks after amiodarone stopped.	\N	FDA label warning	t	17767	11289	\N	ONC_HIGH_PRIORITY	ONC-083	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
110	1191	Aspirin	197381	Ibuprofen	MODERATE	Combined NSAID use increases GI bleeding risk	Avoid combining NSAIDs when possible. Use gastroprotection if necessary.	\N	Well-established additive toxicity	t	1191	197381	\N	ONC_HIGH_PRIORITY	ONC-009-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
111	6916	Metformin	197381	Ibuprofen	MODERATE	NSAIDs may reduce metformin efficacy and increase lactic acidosis risk	Monitor renal function and blood glucose.	\N	Theoretical concern with limited evidence	t	6916	197381	\N	ONC_HIGH_PRIORITY	ONC-010-REV	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
112	5640	Lithium	197381	Ibuprofen	HIGH	NSAIDs decrease lithium clearance by 15-20%	Monitor lithium levels when starting or stopping NSAIDs. Adjust dose as needed.	\N	Documented pharmacokinetic interaction	t	5640	197381	\N	ONC_HIGH_PRIORITY	ONC-011-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
113	197381	Ibuprofen	6813	Methotrexate	HIGH	NSAIDs decrease methotrexate clearance increasing toxicity risk	Avoid NSAIDs during high-dose methotrexate. Monitor for methotrexate toxicity.	\N	Documented pharmacokinetic interaction	t	197381	6813	\N	ONC_HIGH_PRIORITY	ONC-012-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
114	9524	Trimethoprim	6813	Methotrexate	HIGH	Both drugs inhibit folate metabolism leading to severe bone marrow suppression	Avoid combination or use with extreme caution. Monitor CBC frequently.	\N	Additive antifolate toxicity	t	9524	6813	\N	ONC_HIGH_PRIORITY	ONC-013-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
115	8123	Phenytoin	6813	Methotrexate	HIGH	Complex interaction affecting methotrexate and phenytoin levels	Monitor both drug levels and adjust doses as needed.	\N	Documented bidirectional interaction	t	8123	6813	\N	ONC_HIGH_PRIORITY	ONC-014-REV	MODERATE	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
116	38413	Probenecid	6813	Methotrexate	HIGH	Probenecid decreases methotrexate renal clearance	Avoid combination with high-dose methotrexate. Monitor for toxicity.	\N	Pharmacokinetic interaction	t	38413	6813	\N	ONC_HIGH_PRIORITY	ONC-015-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
82	17767	Amiodarone	2551	Digoxin	HIGH	Amiodarone increases digoxin levels by 70-100%	Reduce digoxin dose by 50% when starting amiodarone. Monitor digoxin levels.	\N	Pharmacokinetic interaction	t	17767	2551	\N	ONC_HIGH_PRIORITY	ONC-081	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
118	10600	Verapamil	2551	Digoxin	HIGH	Verapamil increases digoxin levels and additive bradycardia	Reduce digoxin dose by 25-50%. Monitor heart rate and digoxin levels.	\N	Documented interaction with clinical significance	t	10600	2551	\N	ONC_HIGH_PRIORITY	ONC-017-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
119	17128	Diltiazem	2551	Digoxin	HIGH	Diltiazem increases digoxin levels and additive AV nodal effects	Monitor digoxin levels and heart rate. May need dose adjustment.	\N	Similar mechanism to verapamil	t	17128	2551	\N	ONC_HIGH_PRIORITY	ONC-018-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
120	6754	Quinidine	2551	Digoxin	CONTRAINDICATED	Quinidine increases digoxin levels by 50-100% via multiple mechanisms	Reduce digoxin dose by 50%. Monitor digoxin levels and QT interval.	\N	Well-established dangerous interaction	t	6754	2551	\N	ONC_HIGH_PRIORITY	ONC-019-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
121	29046	Spironolactone	2551	Digoxin	MODERATE	Spironolactone may interfere with digoxin assay and renal clearance	Monitor digoxin levels. Be aware of assay interference.	\N	Mixed evidence	t	29046	2551	\N	ONC_HIGH_PRIORITY	ONC-020-REV	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
122	20352	Contrast Media (Iodinated)	6916	Metformin	HIGH	Metformin accumulation may cause lactic acidosis with renal impairment from contrast	Hold metformin before contrast procedure. Resume 48h after if renal function stable.	\N	Guidelines-based recommendation	t	20352	6916	\N	ONC_HIGH_PRIORITY	ONC-021-REV	HIGH	ACR Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
123	82122	Alcohol	6916	Metformin	MODERATE	Alcohol increases lactic acidosis risk with metformin	Avoid excessive alcohol consumption with metformin.	\N	Mechanism-based concern	t	82122	6916	\N	ONC_HIGH_PRIORITY	ONC-022-REV	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
124	1819	Diuretics (Thiazide)	5640	Lithium	HIGH	Thiazides reduce lithium clearance causing toxicity	Monitor lithium levels closely. May need to reduce lithium dose.	\N	Well-documented interaction	t	1819	5640	\N	ONC_HIGH_PRIORITY	ONC-023-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
125	3827	Enalapril	5640	Lithium	HIGH	ACE inhibitors reduce lithium clearance	Monitor lithium levels when starting ACE inhibitors.	\N	Class effect with ACE inhibitors	t	3827	5640	\N	ONC_HIGH_PRIORITY	ONC-024-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
126	52175	Losartan	5640	Lithium	HIGH	ARBs may reduce lithium clearance similar to ACE inhibitors	Monitor lithium levels when starting ARBs.	\N	Class effect expected	t	52175	5640	\N	ONC_HIGH_PRIORITY	ONC-025-REV	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
127	196503	Clarithromycin	36567	Simvastatin	CONTRAINDICATED	Clarithromycin is a strong CYP3A4 inhibitor increasing statin myopathy risk	Avoid combination. Use azithromycin as alternative or suspend statin therapy.	\N	FDA boxed warning for myopathy risk	t	196503	36567	\N	ONC_HIGH_PRIORITY	ONC-026-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
128	28439	Erythromycin	36567	Simvastatin	CONTRAINDICATED	Erythromycin inhibits CYP3A4 increasing statin exposure and myopathy risk	Avoid combination. Consider alternative antibiotic or statin.	\N	Well-documented CYP3A4 interaction	t	28439	36567	\N	ONC_HIGH_PRIORITY	ONC-027-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
100	4337	Fluconazole	36567	Simvastatin	HIGH	Fluconazole inhibits CYP3A4 increasing statin levels	Limit simvastatin dose or use alternative statin.	\N	CYP3A4 interaction	t	4337	36567	\N	ONC_HIGH_PRIORITY	ONC-099	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
83	17767	Amiodarone	36567	Simvastatin	HIGH	Amiodarone inhibits CYP3A4 increasing statin myopathy risk	Limit simvastatin to 10mg daily with amiodarone.	\N	FDA label restriction	t	17767	36567	\N	ONC_HIGH_PRIORITY	ONC-082	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
131	203150	Cyclosporine	36567	Simvastatin	CONTRAINDICATED	Cyclosporine dramatically increases statin levels	Avoid combination. Very high myopathy/rhabdomyolysis risk.	\N	FDA contraindication	t	203150	36567	\N	ONC_HIGH_PRIORITY	ONC-030-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
132	261962	Grapefruit Juice	36567	Simvastatin	HIGH	Grapefruit inhibits intestinal CYP3A4 increasing statin absorption	Avoid grapefruit juice with simvastatin.	\N	Food-drug interaction	t	261962	36567	\N	ONC_HIGH_PRIORITY	ONC-031-REV	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
133	196503	Clarithromycin	83367	Atorvastatin	HIGH	CYP3A4 inhibition increases atorvastatin exposure and myopathy risk	Limit atorvastatin to 20mg daily with clarithromycin. Monitor for muscle symptoms.	\N	FDA labeling restriction	t	196503	83367	\N	ONC_HIGH_PRIORITY	ONC-032-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
134	3640	Cyclosporine	83367	Atorvastatin	CONTRAINDICATED	Cyclosporine significantly increases atorvastatin levels	Limit atorvastatin to 10mg daily with cyclosporine if used.	\N	FDA label restriction	t	3640	83367	\N	ONC_HIGH_PRIORITY	ONC-033-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
135	6470	MAO Inhibitors	4441	Fluoxetine	CONTRAINDICATED	Serotonin syndrome - potentially fatal	Contraindicated. Allow 5 weeks washout between fluoxetine and MAOIs.	\N	Life-threatening interaction	t	6470	4441	\N	ONC_HIGH_PRIORITY	ONC-034-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
67	10689	Tramadol	4441	Fluoxetine	HIGH	Increased serotonin syndrome risk and possible seizures	Use with caution. Monitor for serotonin syndrome symptoms.	\N	Both affect serotonin	t	10689	4441	\N	ONC_HIGH_PRIORITY	ONC-066	MODERATE	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
137	8787	Pimozide	4441	Fluoxetine	CONTRAINDICATED	Fluoxetine inhibits pimozide metabolism increasing QT prolongation risk	Contraindicated combination. Risk of fatal arrhythmia.	\N	FDA contraindication	t	8787	4441	\N	ONC_HIGH_PRIORITY	ONC-036-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
86	35636	Thioridazine	4441	Fluoxetine	CONTRAINDICATED	Fluoxetine increases thioridazine levels causing QT prolongation	Contraindicated. Risk of fatal arrhythmia.	\N	FDA contraindication	t	35636	4441	\N	ONC_HIGH_PRIORITY	ONC-085	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
139	6470	MAO Inhibitors	42347	Sertraline	CONTRAINDICATED	Serotonin syndrome risk	Contraindicated. Allow 2 weeks washout between sertraline and MAOIs.	\N	Life-threatening interaction	t	6470	42347	\N	ONC_HIGH_PRIORITY	ONC-038-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
140	8787	Pimozide	42347	Sertraline	CONTRAINDICATED	Sertraline inhibits pimozide metabolism causing QT prolongation	Contraindicated. Use alternative antipsychotic.	\N	FDA contraindication	t	8787	42347	\N	ONC_HIGH_PRIORITY	ONC-039-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
141	6470	MAO Inhibitors	32937	Paroxetine	CONTRAINDICATED	Serotonin syndrome risk	Contraindicated. Allow 2 weeks washout.	\N	Life-threatening interaction	t	6470	32937	\N	ONC_HIGH_PRIORITY	ONC-040-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
142	8787	Pimozide	32937	Paroxetine	CONTRAINDICATED	Paroxetine inhibits pimozide metabolism	Contraindicated. Risk of fatal arrhythmia.	\N	FDA contraindication	t	8787	32937	\N	ONC_HIGH_PRIORITY	ONC-041-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
87	35636	Thioridazine	32937	Paroxetine	CONTRAINDICATED	Paroxetine increases thioridazine levels	Contraindicated. Risk of fatal arrhythmia.	\N	FDA contraindication	t	35636	32937	\N	ONC_HIGH_PRIORITY	ONC-086	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
144	40790	Omeprazole	114979	Clopidogrel	HIGH	Omeprazole reduces clopidogrel activation via CYP2C19 inhibition	Use pantoprazole instead of omeprazole. Monitor for cardiovascular events.	\N	FDA warning on reduced antiplatelet effect	t	40790	114979	\N	ONC_HIGH_PRIORITY	ONC-043-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
145	28439	Esomeprazole	114979	Clopidogrel	HIGH	Esomeprazole reduces clopidogrel activation similar to omeprazole	Prefer pantoprazole or other alternatives.	\N	Class effect with some PPIs	t	28439	114979	\N	ONC_HIGH_PRIORITY	ONC-044-REV	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
146	4337	Fluconazole	114979	Clopidogrel	HIGH	CYP2C19 inhibition reduces clopidogrel active metabolite formation	Avoid combination if possible. Consider alternative antifungal.	\N	Pharmacokinetic interaction	t	4337	114979	\N	ONC_HIGH_PRIORITY	ONC-045-REV	MODERATE	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
147	196503	Clarithromycin	3640	Cyclosporine	HIGH	CYP3A4 inhibition significantly increases cyclosporine levels	Monitor cyclosporine levels closely. May need 50% dose reduction.	\N	Well-documented nephrotoxicity risk	t	196503	3640	\N	ONC_HIGH_PRIORITY	ONC-046-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
148	1373	Carbamazepine	3640	Cyclosporine	HIGH	Carbamazepine induces CYP3A4 decreasing cyclosporine levels	Monitor cyclosporine levels. May need dose increase.	\N	CYP3A4 induction	t	1373	3640	\N	ONC_HIGH_PRIORITY	ONC-047-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
149	8123	Phenytoin	3640	Cyclosporine	HIGH	Phenytoin induces cyclosporine metabolism	Monitor cyclosporine levels closely during phenytoin therapy.	\N	CYP3A4 induction	t	8123	3640	\N	ONC_HIGH_PRIORITY	ONC-048-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
150	8331	Potassium Chloride	3640	Cyclosporine	HIGH	Cyclosporine causes potassium retention	Monitor potassium levels. Avoid potassium supplements if possible.	\N	Additive hyperkalemia risk	t	8331	3640	\N	ONC_HIGH_PRIORITY	ONC-049-REV	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
151	29046	Spironolactone	8331	Potassium Chloride	HIGH	Additive hyperkalemia risk	Avoid combination or monitor potassium closely. Use lowest effective doses.	\N	Additive effect on potassium	t	29046	8331	\N	ONC_HIGH_PRIORITY	ONC-050-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
152	3827	Enalapril	8331	Potassium Chloride	HIGH	ACE inhibitors reduce potassium excretion	Monitor potassium when using supplements with ACE inhibitors.	\N	Documented additive effect	t	3827	8331	\N	ONC_HIGH_PRIORITY	ONC-051-REV	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
153	52175	Losartan	8331	Potassium Chloride	HIGH	ARBs reduce potassium excretion similar to ACE inhibitors	Monitor potassium with potassium supplements.	\N	Class effect	t	52175	8331	\N	ONC_HIGH_PRIORITY	ONC-052-REV	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
154	3827	Enalapril	29046	Spironolactone	HIGH	Additive hyperkalemia risk from dual RAAS blockade	Monitor potassium and renal function closely. Avoid in high-risk patients.	\N	ONTARGET trial data	t	3827	29046	\N	ONC_HIGH_PRIORITY	ONC-053-REV	HIGH	Clinical Trial	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
155	52175	Losartan	29046	Spironolactone	HIGH	Additive hyperkalemia risk from dual RAAS blockade	Monitor potassium closely. Generally avoid in CKD.	\N	Class effect	t	52175	29046	\N	ONC_HIGH_PRIORITY	ONC-054-REV	HIGH	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
156	6470	MAO Inhibitors	6754	Meperidine	CONTRAINDICATED	Severe potentially fatal reactions including serotonin syndrome and hypertensive crisis	Absolutely contraindicated. Use alternative analgesics.	\N	Life-threatening interaction	t	6470	6754	\N	ONC_HIGH_PRIORITY	ONC-055-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
157	1819	Benzodiazepines	7052	Morphine	HIGH	Additive CNS and respiratory depression risk of death	Avoid combination. If necessary use lowest effective doses and monitor.	\N	FDA boxed warning	t	1819	7052	\N	ONC_HIGH_PRIORITY	ONC-056-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
158	115698	Gabapentin	7052	Morphine	HIGH	Additive CNS depression and respiratory depression	Use lowest doses. Monitor respiratory status.	\N	Emerging evidence of risk	t	115698	7052	\N	ONC_HIGH_PRIORITY	ONC-057-REV	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
159	73178	Pregabalin	7052	Morphine	HIGH	Additive CNS and respiratory depression	Use lowest doses. Monitor closely.	\N	Class effect with gabapentinoids	t	73178	7052	\N	ONC_HIGH_PRIORITY	ONC-058-REV	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
160	1819	Benzodiazepines	5489	Hydrocodone	HIGH	Additive CNS and respiratory depression	Avoid combination. FDA boxed warning.	\N	FDA boxed warning	t	1819	5489	\N	ONC_HIGH_PRIORITY	ONC-059-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
161	115698	Gabapentin	5489	Hydrocodone	HIGH	Additive CNS depression	Use lowest effective doses and monitor.	\N	Gabapentinoid + opioid risk	t	115698	5489	\N	ONC_HIGH_PRIORITY	ONC-060-REV	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
162	1819	Benzodiazepines	7804	Oxycodone	HIGH	Additive CNS and respiratory depression risk of death	Avoid combination. FDA boxed warning on concomitant use.	\N	FDA boxed warning	t	1819	7804	\N	ONC_HIGH_PRIORITY	ONC-061-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
163	115698	Gabapentin	7804	Oxycodone	HIGH	Additive CNS depression	Use lowest doses and monitor.	\N	Gabapentinoid + opioid risk	t	115698	7804	\N	ONC_HIGH_PRIORITY	ONC-062-REV	MODERATE	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
164	1819	Benzodiazepines	3289	Codeine	HIGH	Additive CNS and respiratory depression	Avoid combination or use lowest doses.	\N	FDA boxed warning applies	t	1819	3289	\N	ONC_HIGH_PRIORITY	ONC-063-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
165	6470	MAO Inhibitors	10689	Tramadol	CONTRAINDICATED	Serotonin syndrome risk	Contraindicated. Allow washout period.	\N	Life-threatening interaction	t	6470	10689	\N	ONC_HIGH_PRIORITY	ONC-064-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
166	1819	Benzodiazepines	10689	Tramadol	HIGH	Additive CNS and respiratory depression	Avoid combination. Use lowest doses if necessary.	\N	FDA warning	t	1819	10689	\N	ONC_HIGH_PRIORITY	ONC-065-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
36	4441	Fluoxetine	10689	Tramadol	HIGH	Serotonin syndrome risk plus seizure threshold lowering	Use with caution. Monitor for serotonin syndrome.	\N	Both drugs affect serotonin	t	4441	10689	\N	ONC_HIGH_PRIORITY	ONC-035	MODERATE	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
168	196503	Clarithromycin	8356	Theophylline	HIGH	Clarithromycin inhibits theophylline metabolism	Monitor theophylline levels. May need dose reduction.	\N	CYP3A4 interaction	t	196503	8356	\N	ONC_HIGH_PRIORITY	ONC-067-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
169	28439	Erythromycin	8356	Theophylline	HIGH	Erythromycin inhibits theophylline metabolism	Monitor theophylline levels closely.	\N	Well-documented interaction	t	28439	8356	\N	ONC_HIGH_PRIORITY	ONC-068-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
170	2348	Ciprofloxacin	8356	Theophylline	HIGH	Ciprofloxacin inhibits theophylline metabolism significantly	Reduce theophylline dose by 30-50% or avoid combination.	\N	CYP1A2 inhibition	t	2348	8356	\N	ONC_HIGH_PRIORITY	ONC-069-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
171	1373	Carbamazepine	8356	Theophylline	HIGH	Carbamazepine induces theophylline metabolism	Monitor theophylline levels. May need dose increase.	\N	CYP induction	t	1373	8356	\N	ONC_HIGH_PRIORITY	ONC-070-REV	MODERATE	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
172	11289	Warfarin	1373	Carbamazepine	HIGH	Carbamazepine induces warfarin metabolism	Monitor INR closely. May need warfarin dose increase.	\N	CYP induction	t	11289	1373	\N	ONC_HIGH_PRIORITY	ONC-071-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
78	28728	Valproic Acid	1373	Carbamazepine	HIGH	Complex bidirectional interaction reducing levels of both	Monitor levels of both drugs. Adjust doses as needed.	\N	CYP induction	t	28728	1373	\N	ONC_HIGH_PRIORITY	ONC-077	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
174	196503	Clarithromycin	1373	Carbamazepine	HIGH	Clarithromycin increases carbamazepine levels causing toxicity	Avoid combination or reduce carbamazepine dose.	\N	CYP3A4 inhibition	t	196503	1373	\N	ONC_HIGH_PRIORITY	ONC-073-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
175	28439	Erythromycin	1373	Carbamazepine	HIGH	Erythromycin increases carbamazepine levels	Avoid combination or monitor levels closely.	\N	CYP3A4 inhibition	t	28439	1373	\N	ONC_HIGH_PRIORITY	ONC-074-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
81	8123	Phenytoin	28728	Valproic Acid	HIGH	Complex bidirectional interaction	Monitor levels of both drugs frequently.	\N	Well-documented	t	8123	28728	\N	ONC_HIGH_PRIORITY	ONC-080	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
177	1813	Lamotrigine	28728	Valproic Acid	HIGH	Valproic acid doubles lamotrigine levels	Start lamotrigine at 25mg every other day. Slow titration required.	\N	FDA labeling restriction	t	1813	28728	\N	ONC_HIGH_PRIORITY	ONC-076-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
73	1373	Carbamazepine	28728	Valproic Acid	HIGH	Carbamazepine induces valproic acid metabolism	Monitor valproic acid levels. Dose adjustment needed.	\N	Well-documented interaction	t	1373	28728	\N	ONC_HIGH_PRIORITY	ONC-072	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
4	11289	Warfarin	8123	Phenytoin	HIGH	Complex bidirectional interaction affecting metabolism	Monitor INR and phenytoin levels closely.	\N	Pharmacokinetic interaction documented	t	11289	8123	\N	ONC_HIGH_PRIORITY	ONC-003	HIGH	DrugBank	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
99	4337	Fluconazole	8123	Phenytoin	HIGH	Fluconazole inhibits phenytoin metabolism	Monitor phenytoin levels. May need dose reduction.	\N	CYP2C9 interaction	t	4337	8123	\N	ONC_HIGH_PRIORITY	ONC-098	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
76	28728	Valproic Acid	8123	Phenytoin	HIGH	Complex interaction affecting both drugs	Monitor both drug levels frequently.	\N	Well-documented interaction	t	28728	8123	\N	ONC_HIGH_PRIORITY	ONC-075	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
17	2551	Digoxin	17767	Amiodarone	HIGH	Amiodarone increases digoxin levels by 70-100%	Reduce digoxin dose by 50%. Monitor levels.	\N	Well-documented pharmacokinetic interaction	t	2551	17767	\N	ONC_HIGH_PRIORITY	ONC-016	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
30	36567	Simvastatin	17767	Amiodarone	HIGH	Amiodarone increases statin myopathy risk	Limit simvastatin to 10mg daily.	\N	FDA label restriction	t	36567	17767	\N	ONC_HIGH_PRIORITY	ONC-029	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
9	11289	Warfarin	17767	Amiodarone	CONTRAINDICATED	Amiodarone significantly increases warfarin effect	Reduce warfarin dose by 30-50%. Effect persists for weeks.	\N	FDA label warning	t	11289	17767	\N	ONC_HIGH_PRIORITY	ONC-008	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
185	4441	Fluoxetine	17767	Amiodarone	HIGH	QT prolongation risk from both drugs	Monitor QT interval. Avoid if possible.	\N	Additive QT effect	t	4441	17767	\N	ONC_HIGH_PRIORITY	ONC-084-REV	MODERATE	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
38	4441	Fluoxetine	35636	Thioridazine	CONTRAINDICATED	QT prolongation from CYP2D6 inhibition	Contraindicated. Risk of torsades de pointes.	\N	FDA contraindication	t	4441	35636	\N	ONC_HIGH_PRIORITY	ONC-037	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
43	32937	Paroxetine	35636	Thioridazine	CONTRAINDICATED	QT prolongation from CYP2D6 inhibition	Contraindicated. Risk of fatal arrhythmia.	\N	FDA contraindication	t	32937	35636	\N	ONC_HIGH_PRIORITY	ONC-042	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
188	4337	Fluconazole	35636	Thioridazine	HIGH	QT prolongation from both drugs	Avoid combination due to arrhythmia risk.	\N	Additive QT effect	t	4337	35636	\N	ONC_HIGH_PRIORITY	ONC-087-REV	HIGH	Clinical Guidelines	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
189	196503	Clarithromycin	8787	Pimozide	CONTRAINDICATED	Severe QT prolongation and cardiac arrest risk	Absolutely contraindicated.	\N	FDA contraindication	t	196503	8787	\N	ONC_HIGH_PRIORITY	ONC-088-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
190	28439	Erythromycin	8787	Pimozide	CONTRAINDICATED	QT prolongation risk from CYP3A4 inhibition	Absolutely contraindicated.	\N	FDA contraindication	t	28439	8787	\N	ONC_HIGH_PRIORITY	ONC-089-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
191	4337	Fluconazole	8787	Pimozide	CONTRAINDICATED	QT prolongation from CYP3A4 inhibition	Contraindicated. Use alternative antifungal.	\N	FDA warning	t	4337	8787	\N	ONC_HIGH_PRIORITY	ONC-090-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
192	48937	Colchicine	196503	Clarithromycin	CONTRAINDICATED	Clarithromycin dramatically increases colchicine levels	Contraindicated especially with renal impairment. Fatal cases reported.	\N	FDA warning	t	48937	196503	\N	ONC_HIGH_PRIORITY	ONC-091-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
193	228029	Dronedarone	196503	Clarithromycin	CONTRAINDICATED	CYP3A4 inhibition increases dronedarone levels and QT prolongation	Contraindicated combination.	\N	FDA contraindication	t	228029	196503	\N	ONC_HIGH_PRIORITY	ONC-092-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
194	321988	Ranolazine	196503	Clarithromycin	CONTRAINDICATED	CYP3A4 inhibition significantly increases ranolazine levels	Contraindicated. Risk of QT prolongation.	\N	FDA contraindication	t	321988	196503	\N	ONC_HIGH_PRIORITY	ONC-093-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
195	48937	Colchicine	28439	Erythromycin	HIGH	Erythromycin increases colchicine levels	Reduce colchicine dose. Avoid in renal/hepatic impairment.	\N	CYP3A4 interaction	t	48937	28439	\N	ONC_HIGH_PRIORITY	ONC-094-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
196	321988	Ranolazine	28439	Erythromycin	HIGH	Erythromycin increases ranolazine levels	Limit ranolazine dose. Monitor QT interval.	\N	CYP3A4 interaction	t	321988	28439	\N	ONC_HIGH_PRIORITY	ONC-095-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
197	196503	Cisapride	4337	Fluconazole	CONTRAINDICATED	QT prolongation and torsades de pointes risk	Contraindicated. Cisapride restricted availability.	\N	FDA withdrawal related	t	196503	4337	\N	ONC_HIGH_PRIORITY	ONC-096-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
6	11289	Warfarin	4337	Fluconazole	HIGH	CYP2C9 inhibition increases warfarin effect	Monitor INR closely. Reduce warfarin dose 25-50%.	\N	Well-documented CYP2C9 interaction	t	11289	4337	\N	ONC_HIGH_PRIORITY	ONC-005	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
80	8123	Phenytoin	4337	Fluconazole	HIGH	CYP2C9 inhibition increases phenytoin levels	Monitor phenytoin levels. Adjust dose as needed.	\N	CYP2C9 inhibition	t	8123	4337	\N	ONC_HIGH_PRIORITY	ONC-079	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
29	36567	Simvastatin	4337	Fluconazole	HIGH	CYP3A4 inhibition increases statin levels	Avoid simvastatin or use lowest dose.	\N	CYP3A4 inhibition	t	36567	4337	\N	ONC_HIGH_PRIORITY	ONC-028	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
201	196503	Clarithromycin	203150	Tacrolimus	HIGH	CYP3A4 inhibition dramatically increases tacrolimus levels	Monitor tacrolimus levels closely. Significant dose reduction needed.	\N	Nephrotoxicity risk	t	196503	203150	\N	ONC_HIGH_PRIORITY	ONC-100-REV	HIGH	FDA Label	\N	ONC-2024-Q4	2026-01-20 16:26:13.113206+00	2026-01-20 16:04:00.75147+00
\.


--
-- Data for Name: lab_reference_ranges; Type: TABLE DATA; Schema: public; Owner: -
--

COPY public.lab_reference_ranges (id, loinc_code, component, property, time_aspect, system, scale_type, method_type, class, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours, deprecated, fact_id, source_version, created_at) FROM stdin;
1	2951-2	Sodium	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Sodium SerPl-mCnc	Sodium [Moles/volume] in Serum or Plasma	mmol/L	136.0000	145.0000	120.0000	160.0000	adult	all	electrolyte	Low sodium may indicate SIADH or diuretic use. High sodium indicates dehydration or diabetes insipidus.	10.00	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
2	2823-3	Potassium	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Potassium SerPl-mCnc	Potassium [Moles/volume] in Serum or Plasma	mmol/L	3.5000	5.0000	2.5000	6.5000	adult	all	electrolyte	Monitor for cardiac arrhythmias at extremes. Consider hemolysis artifact if elevated.	20.00	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
3	2075-0	Chloride	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Chloride SerPl-mCnc	Chloride [Moles/volume] in Serum or Plasma	mmol/L	98.0000	106.0000	80.0000	120.0000	adult	all	electrolyte	Interpret with sodium for acid-base status.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
4	1963-8	Bicarbonate	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	HCO3 SerPl-mCnc	Bicarbonate [Moles/volume] in Serum or Plasma	mmol/L	22.0000	29.0000	10.0000	40.0000	adult	all	electrolyte	Low indicates metabolic acidosis. High indicates metabolic alkalosis.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
5	2160-0	Creatinine	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Creat SerPl-mCnc	Creatinine [Mass/volume] in Serum or Plasma	mg/dL	0.7000	1.3000	0.4000	10.0000	adult	male	renal	Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO.	50.00	48	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
6	2160-0	Creatinine	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Creat SerPl-mCnc	Creatinine [Mass/volume] in Serum or Plasma	mg/dL	0.6000	1.1000	0.4000	10.0000	adult	female	renal	Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO.	50.00	48	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
7	3094-0	BUN	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	BUN SerPl-mCnc	Urea nitrogen [Mass/volume] in Serum or Plasma	mg/dL	7.0000	20.0000	2.0000	100.0000	adult	all	renal	BUN:Cr ratio >20 suggests prerenal azotemia.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
8	33914-3	eGFR	ArVRat	Pt	Ser/Plas	Qn	CKD-EPI	CHEM	eGFR CKD-EPI	Glomerular filtration rate/1.73 sq M.predicted by CKD-EPI	mL/min/1.73m2	90.0000	999.0000	15.0000	999.0000	adult	all	renal	Stage CKD: >90=G1 60-89=G2 45-59=G3a 30-44=G3b 15-29=G4 <15=G5.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
9	17861-6	Calcium	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Calcium SerPl-mCnc	Calcium [Mass/volume] in Serum or Plasma	mg/dL	8.6000	10.2000	6.0000	14.0000	adult	all	electrolyte	Correct for albumin: Corrected Ca = measured Ca + 0.8*(4.0 - albumin).	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
10	2000-8	Calcium ionized	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Calcium.ionized SerPl-mCnc	Calcium.ionized [Moles/volume] in Serum or Plasma	mmol/L	1.1200	1.3200	0.8000	1.6000	adult	all	electrolyte	True measure of metabolically active calcium.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
11	2777-1	Phosphorus	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Phos SerPl-mCnc	Phosphate [Mass/volume] in Serum or Plasma	mg/dL	2.5000	4.5000	1.0000	9.0000	adult	all	electrolyte	Inverse relationship with calcium. Monitor in CKD.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
12	19123-9	Magnesium	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Magnesium SerPl-mCnc	Magnesium [Mass/volume] in Serum or Plasma	mg/dL	1.7000	2.2000	1.0000	4.0000	adult	all	electrolyte	Low Mg potentiates digoxin toxicity and refractory hypokalemia.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
13	2345-7	Glucose	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Glucose SerPl-mCnc	Glucose [Mass/volume] in Serum or Plasma	mg/dL	70.0000	100.0000	40.0000	500.0000	adult	all	metabolic	Fasting <100 normal. 100-125 prediabetes. >=126 diabetes (confirm with repeat).	25.00	4	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
14	4548-4	HbA1c	MFr	Pt	Bld	Qn	HPLC	CHEM	HgbA1c MFr Bld	Hemoglobin A1c/Hemoglobin.total in Blood	%	4.0000	5.6000	3.0000	15.0000	adult	all	metabolic	<5.7% normal. 5.7-6.4% prediabetes. >=6.5% diabetes. Target <7% for most diabetics.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
15	2093-3	Cholesterol total	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Cholest SerPl-mCnc	Cholesterol [Mass/volume] in Serum or Plasma	mg/dL	0.0000	200.0000	0.0000	400.0000	adult	all	lipid	Desirable <200. Borderline 200-239. High >=240.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
16	2085-9	HDL Cholesterol	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	HDLc SerPl-mCnc	Cholesterol in HDL [Mass/volume] in Serum or Plasma	mg/dL	40.0000	999.0000	20.0000	999.0000	adult	male	lipid	Low <40 men <50 women is CVD risk factor.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
17	2085-9	HDL Cholesterol	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	HDLc SerPl-mCnc	Cholesterol in HDL [Mass/volume] in Serum or Plasma	mg/dL	50.0000	999.0000	20.0000	999.0000	adult	female	lipid	Low <40 men <50 women is CVD risk factor.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
18	13457-7	LDL Cholesterol calculated	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	LDLc SerPl Calc-mCnc	Cholesterol in LDL [Mass/volume] in Serum or Plasma by calculation	mg/dL	0.0000	100.0000	0.0000	300.0000	adult	all	lipid	Target depends on CV risk. Very high risk <55. High risk <70. Moderate <100.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
19	2571-8	Triglycerides	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Trigl SerPl-mCnc	Triglyceride [Mass/volume] in Serum or Plasma	mg/dL	0.0000	150.0000	0.0000	500.0000	adult	all	lipid	Normal <150. Borderline 150-199. High 200-499. Very high >=500 (pancreatitis risk).	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
20	718-7	Hemoglobin	MCnc	Pt	Bld	Qn	\N	HEM/BC	Hgb Bld-mCnc	Hemoglobin [Mass/volume] in Blood	g/dL	13.5000	17.5000	7.0000	20.0000	adult	male	hematology	Anemia <13 men <12 women. Consider transfusion threshold 7-8 g/dL.	25.00	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
21	718-7	Hemoglobin	MCnc	Pt	Bld	Qn	\N	HEM/BC	Hgb Bld-mCnc	Hemoglobin [Mass/volume] in Blood	g/dL	12.0000	16.0000	7.0000	20.0000	adult	female	hematology	Anemia <13 men <12 women. Consider transfusion threshold 7-8 g/dL.	25.00	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
22	4544-3	Hematocrit	VFr	Pt	Bld	Qn	\N	HEM/BC	Hct VFr Bld	Hematocrit [Volume Fraction] of Blood	%	38.8000	50.0000	20.0000	60.0000	adult	male	hematology	Roughly 3x hemoglobin. Elevated in polycythemia or dehydration.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
23	4544-3	Hematocrit	VFr	Pt	Bld	Qn	\N	HEM/BC	Hct VFr Bld	Hematocrit [Volume Fraction] of Blood	%	34.9000	44.5000	20.0000	60.0000	adult	female	hematology	Roughly 3x hemoglobin. Elevated in polycythemia or dehydration.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
24	777-3	Platelet count	NCnc	Pt	Bld	Qn	\N	HEM/BC	Platelet # Bld	Platelets [#/volume] in Blood	10*3/uL	150.0000	400.0000	50.0000	1000.0000	adult	all	hematology	<100 thrombocytopenia. >50% drop in 5-10 days consider HIT if on heparin.	50.00	120	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
25	6690-2	WBC	NCnc	Pt	Bld	Qn	\N	HEM/BC	WBC # Bld	Leukocytes [#/volume] in Blood	10*3/uL	4.5000	11.0000	1.0000	30.0000	adult	all	hematology	Leukocytosis >11. Leukopenia <4.5. Neutropenia ANC <1500 needs evaluation.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
26	5902-2	PT	Time	Pt	PPP	Qn	\N	COAG	PT PPP	Prothrombin time (PT)	seconds	11.0000	13.5000	9.0000	50.0000	adult	all	coagulation	Monitors warfarin therapy. Elevated with liver disease or vitamin K deficiency.	\N	8	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
27	6301-6	INR	Rto	Pt	Bld	Qn	\N	COAG	INR Bld	INR in Blood by Coagulation assay	\N	0.9000	1.1000	0.8000	6.0000	adult	all	coagulation	Warfarin target 2-3 for most indications. 2.5-3.5 for mechanical valves.	30.00	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
28	3173-2	aPTT	Time	Pt	PPP	Qn	\N	COAG	aPTT PPP	aPTT in Platelet poor plasma by Coagulation assay	seconds	25.0000	35.0000	20.0000	100.0000	adult	all	coagulation	Monitors heparin. Prolonged with lupus anticoagulant or factor deficiencies.	\N	8	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
29	14979-9	aPTT ratio	Rto	Pt	PPP	Qn	\N	COAG	aPTT ratio PPP	aPTT ratio in Platelet poor plasma	\N	0.9000	1.2000	0.5000	4.0000	adult	all	coagulation	Heparin therapeutic range typically 1.5-2.5x control.	\N	8	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
30	2532-0	LDH	CCnc	Pt	Ser/Plas	Qn	\N	CHEM	LDH SerPl-cCnc	Lactate dehydrogenase [Enzymatic activity/volume] in Serum or Plasma	U/L	140.0000	280.0000	50.0000	2000.0000	adult	all	enzyme	Elevated in hemolysis tissue damage MI PE. Nonspecific but sensitive.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
31	1742-6	ALT	CCnc	Pt	Ser/Plas	Qn	\N	CHEM	ALT SerPl-cCnc	Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma	U/L	7.0000	56.0000	5.0000	1000.0000	adult	all	liver	More liver-specific than AST. >3x ULN significant. >10x ULN acute hepatitis.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
32	1920-8	AST	CCnc	Pt	Ser/Plas	Qn	\N	CHEM	AST SerPl-cCnc	Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma	U/L	10.0000	40.0000	5.0000	1000.0000	adult	all	liver	Also elevated in muscle injury MI. AST:ALT >2 suggests alcoholic liver disease.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
33	6768-6	ALP	CCnc	Pt	Ser/Plas	Qn	\N	CHEM	ALP SerPl-cCnc	Alkaline phosphatase [Enzymatic activity/volume] in Serum or Plasma	U/L	44.0000	147.0000	20.0000	1000.0000	adult	all	liver	Elevated in cholestasis bone disease. GGT helps distinguish hepatic source.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
34	2324-2	GGT	CCnc	Pt	Ser/Plas	Qn	\N	CHEM	GGT SerPl-cCnc	Gamma glutamyl transferase [Enzymatic activity/volume] in Serum or Plasma	U/L	9.0000	48.0000	5.0000	500.0000	adult	all	liver	Sensitive for hepatobiliary disease. Elevated by alcohol and many drugs.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
35	1975-2	Total Bilirubin	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Bilirub SerPl-mCnc	Bilirubin.total [Mass/volume] in Serum or Plasma	mg/dL	0.1000	1.2000	0.1000	15.0000	adult	all	liver	Jaundice visible >2.5. Conjugated vs unconjugated guides differential.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
36	1968-7	Direct Bilirubin	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Bilirub Direct SerPl-mCnc	Bilirubin.direct [Mass/volume] in Serum or Plasma	mg/dL	0.0000	0.3000	0.0000	10.0000	adult	all	liver	>50% of total suggests hepatocellular or cholestatic disease.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
37	1751-7	Albumin	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Albumin SerPl-mCnc	Albumin [Mass/volume] in Serum or Plasma	g/dL	3.5000	5.0000	1.5000	6.0000	adult	all	protein	Low in malnutrition liver disease nephrotic syndrome. Half-life ~21 days.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
38	2885-2	Total Protein	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Prot SerPl-mCnc	Protein [Mass/volume] in Serum or Plasma	g/dL	6.0000	8.3000	4.0000	12.0000	adult	all	protein	Globulin gap = Total protein - Albumin. Elevated gap suggests inflammation or myeloma.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
39	33762-6	NT-proBNP	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	NT-proBNP SerPl-mCnc	Natriuretic peptide.B prohormone N-Terminal [Mass/volume] in Serum or Plasma	pg/mL	0.0000	125.0000	0.0000	30000.0000	adult	all	cardiac	Age-adjusted cutoffs: <50yo <450. 50-75yo <900. >75yo <1800. Rules out HF if normal.	50.00	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
40	10839-9	Troponin I	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Troponin I SerPl-mCnc	Troponin I.cardiac [Mass/volume] in Serum or Plasma	ng/mL	0.0000	0.0400	0.0000	50.0000	adult	all	cardiac	99th percentile is cutoff for MI. Serial measurements q3-6h. Rise and fall pattern.	100.00	6	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
41	6598-7	Troponin T	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	Troponin T SerPl-mCnc	Troponin T.cardiac [Mass/volume] in Serum or Plasma	ng/mL	0.0000	0.0100	0.0000	10.0000	adult	all	cardiac	High-sensitivity assay improves early detection. Elevated in CKD non-MI causes.	100.00	6	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
42	30341-2	ESR	Vel	Pt	Bld	Qn	Westergren	CHEM	ESR Bld Qn Westrgrn	Erythrocyte sedimentation rate by Westergren method	mm/h	0.0000	20.0000	0.0000	100.0000	adult	male	inflammatory	Nonspecific inflammation marker. Age-adjusted: Men (age/2) Women (age+10)/2.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
43	30341-2	ESR	Vel	Pt	Bld	Qn	Westergren	CHEM	ESR Bld Qn Westrgrn	Erythrocyte sedimentation rate by Westergren method	mm/h	0.0000	30.0000	0.0000	100.0000	adult	female	inflammatory	Nonspecific inflammation marker. Age-adjusted: Men (age/2) Women (age+10)/2.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
44	1988-5	CRP	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	CRP SerPl-mCnc	C reactive protein [Mass/volume] in Serum or Plasma	mg/L	0.0000	3.0000	0.0000	200.0000	adult	all	inflammatory	Acute phase reactant. >10 suggests infection or inflammation. hs-CRP for CV risk.	\N	24	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
45	3016-3	TSH	ACnc	Pt	Ser/Plas	Qn	\N	CHEM	TSH SerPl-aCnc	Thyrotropin [Units/volume] in Serum or Plasma	mIU/L	0.4000	4.0000	0.0100	100.0000	adult	all	endocrine	Low TSH + high T4 = hyperthyroid. High TSH + low T4 = hypothyroid. Screen first.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
46	3026-2	Free T4	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	T4 Free SerPl-mCnc	Thyroxine (T4) free [Mass/volume] in Serum or Plasma	ng/dL	0.8000	1.8000	0.4000	7.0000	adult	all	endocrine	Free T4 preferred over total T4. Interpret with TSH.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
47	3051-0	Free T3	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	T3 Free SerPl-mCnc	Triiodothyronine (T3) free [Mass/volume] in Serum or Plasma	pg/mL	2.3000	4.2000	1.0000	10.0000	adult	all	endocrine	May be elevated in early hyperthyroidism before T4 rises.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
48	2458-8	IgA	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	IgA SerPl-mCnc	IgA [Mass/volume] in Serum or Plasma	mg/dL	70.0000	400.0000	20.0000	800.0000	adult	all	immunology	Deficiency common (1:500). May have false negative celiac serology if IgA deficient.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
49	2465-3	IgG	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	IgG SerPl-mCnc	IgG [Mass/volume] in Serum or Plasma	mg/dL	700.0000	1600.0000	200.0000	4000.0000	adult	all	immunology	Low in immunodeficiency or protein loss. High in chronic infection or myeloma.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
50	2472-9	IgM	MCnc	Pt	Ser/Plas	Qn	\N	CHEM	IgM SerPl-mCnc	IgM [Mass/volume] in Serum or Plasma	mg/dL	40.0000	230.0000	20.0000	500.0000	adult	all	immunology	First antibody in acute infection. Elevated in Waldenstrom macroglobulinemia.	\N	0	f	\N	LOINC-2024	2026-01-20 16:26:13.113206+00
\.


--
-- Name: formulary_coverage_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.formulary_coverage_id_seq', 328, true);


--
-- Name: ingestion_metadata_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.ingestion_metadata_id_seq', 6, true);


--
-- Name: interaction_matrix_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.interaction_matrix_id_seq', 401, true);


--
-- Name: lab_reference_ranges_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.lab_reference_ranges_id_seq', 100, true);


--
-- Name: formulary_coverage formulary_coverage_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.formulary_coverage
    ADD CONSTRAINT formulary_coverage_pkey PRIMARY KEY (id);


--
-- Name: ingestion_metadata ingestion_metadata_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ingestion_metadata
    ADD CONSTRAINT ingestion_metadata_pkey PRIMARY KEY (id);


--
-- Name: interaction_matrix interaction_matrix_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interaction_matrix
    ADD CONSTRAINT interaction_matrix_pkey PRIMARY KEY (id);


--
-- Name: lab_reference_ranges lab_reference_ranges_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_reference_ranges
    ADD CONSTRAINT lab_reference_ranges_pkey PRIMARY KEY (id);


--
-- Name: formulary_coverage uq_formulary_entry; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.formulary_coverage
    ADD CONSTRAINT uq_formulary_entry UNIQUE (contract_id, plan_id, rxcui, ndc, effective_year);


--
-- Name: ingestion_metadata uq_ingestion; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.ingestion_metadata
    ADD CONSTRAINT uq_ingestion UNIQUE (source_name, source_version, load_timestamp);


--
-- Name: interaction_matrix uq_interaction_pair; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interaction_matrix
    ADD CONSTRAINT uq_interaction_pair UNIQUE (drug1_rxcui, drug2_rxcui, source_dataset);


--
-- Name: lab_reference_ranges uq_lab_range; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_reference_ranges
    ADD CONSTRAINT uq_lab_range UNIQUE (loinc_code, age_group, sex, source_version);


--
-- Name: idx_formulary_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_formulary_lookup ON public.formulary_coverage USING btree (rxcui, contract_id, plan_id, effective_year);


--
-- Name: idx_formulary_plan; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_formulary_plan ON public.formulary_coverage USING btree (contract_id, plan_id);


--
-- Name: idx_formulary_rxcui; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_formulary_rxcui ON public.formulary_coverage USING btree (rxcui);


--
-- Name: idx_formulary_tier; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_formulary_tier ON public.formulary_coverage USING btree (tier_level_code);


--
-- Name: idx_formulary_year; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_formulary_year ON public.formulary_coverage USING btree (effective_year);


--
-- Name: idx_interaction_drug1; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_interaction_drug1 ON public.interaction_matrix USING btree (drug1_rxcui);


--
-- Name: idx_interaction_drug2; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_interaction_drug2 ON public.interaction_matrix USING btree (drug2_rxcui);


--
-- Name: idx_interaction_lookup; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_interaction_lookup ON public.interaction_matrix USING btree (drug1_rxcui, drug2_rxcui, severity);


--
-- Name: idx_interaction_pair; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_interaction_pair ON public.interaction_matrix USING btree (drug1_rxcui, drug2_rxcui);


--
-- Name: idx_interaction_severity; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_interaction_severity ON public.interaction_matrix USING btree (severity);


--
-- Name: idx_interaction_source; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_interaction_source ON public.interaction_matrix USING btree (source_dataset);


--
-- Name: idx_lab_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_lab_category ON public.lab_reference_ranges USING btree (clinical_category);


--
-- Name: idx_lab_component; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_lab_component ON public.lab_reference_ranges USING btree (component);


--
-- Name: idx_lab_delta; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_lab_delta ON public.lab_reference_ranges USING btree (loinc_code) WHERE (delta_check_percent IS NOT NULL);


--
-- Name: idx_lab_loinc; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_lab_loinc ON public.lab_reference_ranges USING btree (loinc_code);


--
-- Name: formulary_coverage trg_protect_formulary_coverage; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_protect_formulary_coverage BEFORE INSERT OR DELETE OR UPDATE ON public.formulary_coverage FOR EACH ROW WHEN ((CURRENT_USER <> ALL (ARRAY['kb_admin'::name, 'kb_ingest_svc'::name]))) EXECUTE FUNCTION public.prevent_projection_write();


--
-- Name: interaction_matrix trg_protect_interaction_matrix; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_protect_interaction_matrix BEFORE INSERT OR DELETE OR UPDATE ON public.interaction_matrix FOR EACH ROW WHEN ((CURRENT_USER <> ALL (ARRAY['kb_admin'::name, 'kb_ingest_svc'::name]))) EXECUTE FUNCTION public.prevent_projection_write();


--
-- Name: lab_reference_ranges trg_protect_lab_reference_ranges; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_protect_lab_reference_ranges BEFORE INSERT OR DELETE OR UPDATE ON public.lab_reference_ranges FOR EACH ROW WHEN ((CURRENT_USER <> ALL (ARRAY['kb_admin'::name, 'kb_ingest_svc'::name]))) EXECUTE FUNCTION public.prevent_projection_write();


--
-- Name: formulary_coverage formulary_coverage_fact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.formulary_coverage
    ADD CONSTRAINT formulary_coverage_fact_id_fkey FOREIGN KEY (fact_id) REFERENCES public.clinical_facts(fact_id);


--
-- Name: interaction_matrix interaction_matrix_fact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.interaction_matrix
    ADD CONSTRAINT interaction_matrix_fact_id_fkey FOREIGN KEY (fact_id) REFERENCES public.clinical_facts(fact_id);


--
-- Name: lab_reference_ranges lab_reference_ranges_fact_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.lab_reference_ranges
    ADD CONSTRAINT lab_reference_ranges_fact_id_fkey FOREIGN KEY (fact_id) REFERENCES public.clinical_facts(fact_id);


--
-- PostgreSQL database dump complete
--

\unrestrict N6TfcGjjyjKDqqii1xnd2rch6jpdbJTenFTxTSrTtf4SM8FcapKyOmKxv7rJZGO

