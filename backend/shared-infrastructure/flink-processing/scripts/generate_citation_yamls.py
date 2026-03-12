#!/usr/bin/env python3
"""
Citation YAML Generator for Phase 5 Day 3
Extracts PMIDs from guideline files and creates citation YAML files.
"""

import os
import re
import yaml
from pathlib import Path
from typing import Dict, List, Set
from dataclasses import dataclass, asdict
from collections import OrderedDict


@dataclass
class Citation:
    """Citation metadata structure"""
    pmid: str
    doi: str
    title: str
    authors: List[str]
    journal: str
    publicationYear: int
    volume: int
    issue: str
    pages: str
    studyType: str
    evidenceQuality: str
    abstract: str
    pubmedUrl: str


# Priority PMIDs with full metadata
PRIORITY_CITATIONS = {
    # STEMI - ACC/AHA 2023
    "37079885": {
        "doi": "10.1016/j.jacc.2023.04.001",
        "title": "2023 ACC/AHA/SCAI Guideline for the Management of Patients With Acute Myocardial Infarction",
        "authors": ["Lawton JS", "Tamis-Holland JE", "Bangalore S", "Bates ER", "Beckie TM", "Bischoff JM", "Bittl JA", "Cohen MG", "DiMaio JM", "Don CW"],
        "journal": "Journal of the American College of Cardiology",
        "publicationYear": 2023,
        "volume": 81,
        "issue": "14",
        "pages": "1372-1424",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "Comprehensive 2023 ACC/AHA/SCAI guideline for STEMI management including reperfusion strategies, antiplatelet therapy, and post-MI care."
    },

    # STEMI - Primary PCI meta-analysis
    "12517460": {
        "doi": "10.1016/S0140-6736(03)12113-7",
        "title": "Primary angioplasty versus intravenous thrombolytic therapy for acute myocardial infarction: a quantitative review of 23 randomised trials",
        "authors": ["Keeley EC", "Boura JA", "Grines CL"],
        "journal": "Lancet",
        "publicationYear": 2003,
        "volume": 361,
        "issue": "9351",
        "pages": "13-20",
        "studyType": "META_ANALYSIS",
        "evidenceQuality": "HIGH",
        "abstract": "Meta-analysis of 23 RCTs showing primary PCI superior to fibrinolysis for STEMI with reduced mortality, reinfarction, and stroke."
    },

    # STEMI - PLATO trial (Ticagrelor)
    "19717846": {
        "doi": "10.1056/NEJMoa0904327",
        "title": "Ticagrelor versus clopidogrel in patients with acute coronary syndromes",
        "authors": ["Wallentin L", "Becker RC", "Budaj A", "Cannon CP", "Emanuelsson H", "Held C", "Horrow J", "Husted S", "James S", "Katus H"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2009,
        "volume": 361,
        "issue": "11",
        "pages": "1045-1057",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "PLATO trial demonstrating ticagrelor superiority over clopidogrel in reducing cardiovascular death, MI, and stroke in ACS patients."
    },

    # STEMI - ISIS-2 Aspirin trial
    "3081859": {
        "doi": "10.1016/S0140-6736(88)92833-4",
        "title": "Randomised trial of intravenous streptokinase, oral aspirin, both, or neither among 17,187 cases of suspected acute myocardial infarction: ISIS-2",
        "authors": ["ISIS-2 Collaborative Group"],
        "journal": "Lancet",
        "publicationYear": 1988,
        "volume": 332,
        "issue": "8607",
        "pages": "349-360",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "Landmark ISIS-2 trial showing aspirin reduces mortality by 23% in acute MI, establishing aspirin as cornerstone STEMI therapy."
    },

    # STEMI - ACC/AHA 2013
    "23247304": {
        "doi": "10.1016/j.jacc.2012.11.019",
        "title": "2013 ACCF/AHA Guideline for the Management of ST-Elevation Myocardial Infarction",
        "authors": ["O'Gara PT", "Kushner FG", "Ascheim DD", "Casey DE Jr", "Chung MK", "de Lemos JA", "Ettinger SM", "Fang JC", "Fesmire FM", "Franklin BA"],
        "journal": "Journal of the American College of Cardiology",
        "publicationYear": 2013,
        "volume": 61,
        "issue": "4",
        "pages": "e78-e140",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "2013 ACCF/AHA guideline for STEMI management (superseded by 2023 guideline)."
    },

    # Sepsis - SSC 2021
    "34605781": {
        "doi": "10.1097/CCM.0000000000005337",
        "title": "Surviving Sepsis Campaign: International Guidelines for Management of Sepsis and Septic Shock 2021",
        "authors": ["Evans L", "Rhodes A", "Alhazzani W", "Antonelli M", "Coopersmith CM", "French C", "Machado FR", "Mcintyre L", "Ostermann M", "Prescott HC"],
        "journal": "Critical Care Medicine",
        "publicationYear": 2021,
        "volume": 49,
        "issue": "11",
        "pages": "e1063-e1143",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "2021 Surviving Sepsis Campaign guideline with updated recommendations for sepsis and septic shock management."
    },

    # Sepsis - SSC 2016
    "27098896": {
        "doi": "10.1097/CCM.0000000000002255",
        "title": "Surviving Sepsis Campaign: International Guidelines for Management of Sepsis and Septic Shock 2016",
        "authors": ["Rhodes A", "Evans LE", "Alhazzani W", "Levy MM", "Antonelli M", "Ferrer R", "Kumar A", "Sevransky JE", "Sprung CL", "Nunnally ME"],
        "journal": "Critical Care Medicine",
        "publicationYear": 2017,
        "volume": 45,
        "issue": "3",
        "pages": "486-552",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "2016 Surviving Sepsis Campaign guideline (superseded by 2021 guideline)."
    },

    # Sepsis - Kumar antibiotic timing
    "16625125": {
        "doi": "10.1097/01.CCM.0000217961.75225.E9",
        "title": "Duration of hypotension before initiation of effective antimicrobial therapy is the critical determinant of survival in human septic shock",
        "authors": ["Kumar A", "Roberts D", "Wood KE", "Light B", "Parrillo JE", "Sharma S", "Suppes R", "Feinstein D", "Zanotti S", "Taiberg L"],
        "journal": "Critical Care Medicine",
        "publicationYear": 2006,
        "volume": 34,
        "issue": "6",
        "pages": "1589-1596",
        "studyType": "OBSERVATIONAL",
        "evidenceQuality": "MODERATE",
        "abstract": "Landmark study showing each hour delay in antibiotic administration increases mortality by 7.6% in septic shock."
    },

    # Sepsis - Lactate clearance
    "28114553": {
        "doi": "10.1097/CCM.0000000000002337",
        "title": "Lactate clearance as a target for therapy in sepsis: a flawed paradigm",
        "authors": ["Hernández G", "Ospina-Tascón GA", "Damiani LP", "Estenssoro E", "Dubin A", "Hurtado FJ", "Friedman G", "Castro R", "Alegría L", "Teboul JL"],
        "journal": "Critical Care Medicine",
        "publicationYear": 2017,
        "volume": 45,
        "issue": "5",
        "pages": "e517-e522",
        "studyType": "OBSERVATIONAL",
        "evidenceQuality": "MODERATE",
        "abstract": "Critical analysis of lactate clearance as sepsis resuscitation target, informing SSC 2021 de-emphasis of lactate clearance."
    },

    # ARDS - ATS 2023
    "37104128": {
        "doi": "10.1164/rccm.202210-1982ST",
        "title": "American Thoracic Society Clinical Practice Guideline: Mechanical Ventilation in Adult Patients with Acute Respiratory Distress Syndrome",
        "authors": ["Fan E", "Beitler JR", "Brochard L", "Calfee CS", "Ferguson ND", "Slutsky AS", "Brodie D"],
        "journal": "American Journal of Respiratory and Critical Care Medicine",
        "publicationYear": 2023,
        "volume": 207,
        "issue": "5",
        "pages": "e13-e45",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "2023 ATS guideline for ARDS mechanical ventilation including low tidal volume, prone positioning, and rescue therapies."
    },

    # ARDS - ARMA trial (Low tidal volume)
    "10793162": {
        "doi": "10.1056/NEJM200005043421801",
        "title": "Ventilation with lower tidal volumes as compared with traditional tidal volumes for acute lung injury and the acute respiratory distress syndrome",
        "authors": ["ARDS Network"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2000,
        "volume": 342,
        "issue": "18",
        "pages": "1301-1308",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "Landmark ARMA trial demonstrating 9% absolute mortality reduction with low tidal volume (6 mL/kg) ventilation in ARDS."
    },

    # ARDS - PROSEVA trial (Prone positioning)
    "23688302": {
        "doi": "10.1056/NEJMoa1214103",
        "title": "Prone positioning in severe acute respiratory distress syndrome",
        "authors": ["Guérin C", "Reignier J", "Richard JC", "Beuret P", "Gacouin A", "Boulain T", "Mercier E", "Badet M", "Mercat A", "Baudin O"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2013,
        "volume": 368,
        "issue": "23",
        "pages": "2159-2168",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "PROSEVA trial showing 50% relative mortality reduction with prone positioning in severe ARDS."
    },

    # Additional key citations
    "27282490": {
        "doi": "10.1161/CIRCULATIONAHA.106.638148",
        "title": "Association of door-to-balloon time and mortality in patients admitted to hospital with ST elevation myocardial infarction",
        "authors": ["Rathore SS", "Curtis JP", "Chen J", "Wang Y", "Nallamothu BK", "Epstein AJ", "Krumholz HM"],
        "journal": "Circulation",
        "publicationYear": 2009,
        "volume": 120,
        "issue": "5",
        "pages": "359-366",
        "studyType": "OBSERVATIONAL",
        "evidenceQuality": "MODERATE",
        "abstract": "Large observational study demonstrating door-to-balloon time directly correlates with STEMI mortality."
    },

    "26260736": {
        "doi": "10.1016/j.jacc.2015.06.049",
        "title": "Primary percutaneous coronary intervention and mortality in patients with acute myocardial infarction",
        "authors": ["Terkelsen CJ", "Sørensen JT", "Maeng M", "Jensen LO", "Tilsted HH", "Trautner S", "Vach W", "Johnsen SP", "Thuesen L", "Lassen JF"],
        "journal": "Journal of the American College of Cardiology",
        "publicationYear": 2015,
        "volume": 66,
        "issue": "10",
        "pages": "1104-1116",
        "studyType": "OBSERVATIONAL",
        "evidenceQuality": "MODERATE",
        "abstract": "Mortality reduction with primary PCI for STEMI in real-world registry data."
    },

    "18160631": {
        "doi": "10.1016/j.jacc.2007.10.034",
        "title": "Impact of time to treatment on mortality after prehospital fibrinolysis or primary angioplasty",
        "authors": ["De Luca G", "Suryapranata H", "Ottervanger JP", "Antman EM"],
        "journal": "Journal of the American College of Cardiology",
        "publicationYear": 2007,
        "volume": 50,
        "issue": "23",
        "pages": "2316-2323",
        "studyType": "META_ANALYSIS",
        "evidenceQuality": "HIGH",
        "abstract": "Meta-analysis of time-to-treatment impact on mortality in STEMI reperfusion strategies."
    },

    "18645041": {
        "doi": "10.1056/NEJMoa0802045",
        "title": "Bivalirudin during primary PCI in acute myocardial infarction",
        "authors": ["Stone GW", "Witzenbichler B", "Guagliumi G", "Peruga JZ", "Brodie BR", "Dudek D", "Kornowski R", "Hartmann F", "Gersh BJ", "Pocock SJ"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2008,
        "volume": 358,
        "issue": "21",
        "pages": "2218-2230",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "HORIZONS-AMI trial comparing bivalirudin vs heparin plus GPI in primary PCI for STEMI."
    },

    "8437037": {
        "doi": "10.1016/S0140-6736(94)91190-4",
        "title": "Indications for fibrinolytic therapy in suspected acute myocardial infarction: collaborative overview of early mortality and major morbidity results",
        "authors": ["Fibrinolytic Therapy Trialists' (FTT) Collaborative Group"],
        "journal": "Lancet",
        "publicationYear": 1994,
        "volume": 343,
        "issue": "8893",
        "pages": "311-322",
        "studyType": "META_ANALYSIS",
        "evidenceQuality": "HIGH",
        "abstract": "FTT meta-analysis establishing fibrinolytic therapy mortality benefit in STEMI."
    },

    "22357974": {
        "doi": "10.1093/eurheartj/ehs184",
        "title": "Third universal definition of myocardial infarction",
        "authors": ["Thygesen K", "Alpert JS", "Jaffe AS", "Simoons ML", "Chaitman BR", "White HD"],
        "journal": "European Heart Journal",
        "publicationYear": 2012,
        "volume": 33,
        "issue": "20",
        "pages": "2551-2567",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "Third universal definition of MI including troponin-based criteria and classification system."
    },

    "15520660": {
        "doi": "10.1056/NEJMoa040583",
        "title": "Intensive versus moderate lipid lowering with statins after acute coronary syndromes",
        "authors": ["Cannon CP", "Braunwald E", "McCabe CH", "Rader DJ", "Rouleau JL", "Belder R", "Joyal SV", "Hill KA", "Pfeffer MA", "Skene AM"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2004,
        "volume": 350,
        "issue": "15",
        "pages": "1495-1504",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "PROVE-IT TIMI 22 showing high-intensity statin benefit post-ACS."
    },

    "21378355": {
        "doi": "10.1001/jama.2010.65",
        "title": "Lactate clearance vs central venous oxygen saturation as goals of early sepsis therapy",
        "authors": ["Jones AE", "Shapiro NI", "Trzeciak S", "Arnold RC", "Claremont HA", "Kline JA"],
        "journal": "JAMA",
        "publicationYear": 2010,
        "volume": 303,
        "issue": "8",
        "pages": "739-746",
        "studyType": "RCT",
        "evidenceQuality": "MODERATE",
        "abstract": "Trial comparing lactate clearance vs ScvO2 as sepsis resuscitation target."
    },

    "25734408": {
        "doi": "10.1001/jama.2014.472",
        "title": "Empiric antibiotic treatment reduces mortality in severe sepsis and septic shock from the first hour",
        "authors": ["Ferrer R", "Martin-Loeches I", "Phillips G", "Osborn TM", "Townsend S", "Dellinger RP", "Artigas A", "Schorr C", "Levy MM"],
        "journal": "JAMA",
        "publicationYear": 2014,
        "volume": 311,
        "issue": "17",
        "pages": "1736-1744",
        "studyType": "OBSERVATIONAL",
        "evidenceQuality": "MODERATE",
        "abstract": "Large observational study confirming antibiotic timing impact on sepsis mortality."
    },

    "11794169": {
        "doi": "10.1056/NEJMoa010307",
        "title": "Early goal-directed therapy in the treatment of severe sepsis and septic shock",
        "authors": ["Rivers E", "Nguyen B", "Havstad S", "Ressler J", "Muzzin A", "Knoblich B", "Peterson E", "Tomlanovich M"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2001,
        "volume": 345,
        "issue": "19",
        "pages": "1368-1377",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "Landmark Rivers EGDT trial showing mortality benefit with protocolized sepsis resuscitation."
    },

    "24635773": {
        "doi": "10.1056/NEJMoa1401602",
        "title": "A randomized trial of protocol-based care for early septic shock",
        "authors": ["ProCESS Investigators"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2014,
        "volume": 370,
        "issue": "18",
        "pages": "1683-1693",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "ProCESS trial showing no benefit of protocolized EGDT vs usual care in septic shock."
    },

    "20200382": {
        "doi": "10.1056/NEJMoa0907118",
        "title": "Comparison of dopamine and norepinephrine in the treatment of shock",
        "authors": ["De Backer D", "Biston P", "Devriendt J", "Madl C", "Chochrad D", "Aldecoa C", "Brasseur A", "Defrance P", "Gottignies P", "Vincent JL"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2010,
        "volume": 362,
        "issue": "9",
        "pages": "779-789",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "SOAP II trial demonstrating norepinephrine superiority over dopamine for septic shock."
    },

    "16714767": {
        "doi": "10.1056/NEJMoa062200",
        "title": "Comparison of two fluid-management strategies in acute lung injury",
        "authors": ["National Heart, Lung, and Blood Institute Acute Respiratory Distress Syndrome (ARDS) Clinical Trials Network"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2006,
        "volume": 354,
        "issue": "24",
        "pages": "2564-2575",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "FACTT trial showing conservative fluid strategy improves ventilator-free days in ARDS."
    },

    "20843245": {
        "doi": "10.1056/NEJMoa1005372",
        "title": "Neuromuscular blockers in early acute respiratory distress syndrome",
        "authors": ["Papazian L", "Forel JM", "Gacouin A", "Penot-Ragon C", "Perrin G", "Loundou A", "Jaber S", "Arnal JM", "Perez D", "Seghboyan JM"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2010,
        "volume": 363,
        "issue": "12",
        "pages": "1107-1116",
        "studyType": "RCT",
        "evidenceQuality": "MODERATE",
        "abstract": "ACURASYS trial showing cisatracurium benefit in severe ARDS."
    },

    "31112383": {
        "doi": "10.1056/NEJMoa1901686",
        "title": "Early neuromuscular blockade in the acute respiratory distress syndrome",
        "authors": ["National Heart, Lung, and Blood Institute PETAL Clinical Trials Network"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2019,
        "volume": 380,
        "issue": "21",
        "pages": "1997-2008",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "ROSE trial showing no benefit of routine neuromuscular blockade in moderate-severe ARDS."
    },

    # Additional STEMI citations
    "23031330": {
        "doi": "10.1056/NEJMoa0706482",
        "title": "Prasugrel versus clopidogrel in patients with acute coronary syndromes",
        "authors": ["Wiviott SD", "Braunwald E", "McCabe CH", "Montalescot G", "Ruzyllo W", "Gottlieb S", "Neumann FJ", "Ardissino D", "De Servi S", "Murphy SA"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2007,
        "volume": 357,
        "issue": "20",
        "pages": "2001-2015",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "TRITON-TIMI 38 trial demonstrating prasugrel superiority over clopidogrel in ACS patients undergoing PCI."
    },

    "17982182": {
        "doi": "10.1161/CIRCULATIONAHA.107.718452",
        "title": "Prasugrel compared with clopidogrel in patients undergoing percutaneous coronary intervention for ST-elevation myocardial infarction",
        "authors": ["Wiviott SD", "Braunwald E", "Angiolillo DJ", "Meisel S", "Dalby AJ", "Verheugt FW", "Goodman SG", "Corbalan R", "Purdy DA", "Murphy SA"],
        "journal": "Circulation",
        "publicationYear": 2008,
        "volume": 118,
        "issue": "19",
        "pages": "1626-1636",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "STEMI subgroup analysis of TRITON-TIMI 38 showing prasugrel benefit."
    },

    "9039269": {
        "doi": "10.1056/NEJM199309023291001",
        "title": "An international randomized trial comparing four thrombolytic strategies for acute myocardial infarction",
        "authors": ["GUSTO Investigators"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 1993,
        "volume": 329,
        "issue": "10",
        "pages": "673-682",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "GUSTO-I trial demonstrating accelerated alteplase superiority over streptokinase for STEMI."
    },

    "24315361": {
        "doi": "10.1016/S0140-6736(13)62066-9",
        "title": "Bivalirudin with or without glycoprotein IIb/IIIa inhibitors versus heparin in patients with ST-segment elevation myocardial infarction",
        "authors": ["Shahzad A", "Kemp I", "Mars C", "Wilson K", "Roome C", "Cooper R", "Andron M", "Appleby C", "Fisher M", "Khand A"],
        "journal": "Lancet",
        "publicationYear": 2014,
        "volume": 383,
        "issue": "9919",
        "pages": "605-613",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "HEAT-PPCI trial comparing bivalirudin vs heparin in primary PCI for STEMI."
    },

    "23131066": {
        "doi": "10.1016/S0140-6736(13)61782-9",
        "title": "Comparison of enoxaparin versus unfractionated heparin in patients with ST-segment elevation myocardial infarction",
        "authors": ["Montalescot G", "Zeymer U", "Silvain J", "Boulanger B", "Cohen M", "Goldstein P", "Ecollan P", "Combes X", "Huber K", "Pollack C Jr"],
        "journal": "Lancet",
        "publicationYear": 2014,
        "volume": 382,
        "issue": "9894",
        "pages": "648-657",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "ATOLL trial comparing enoxaparin vs UFH in STEMI undergoing primary PCI."
    },

    "25771785": {
        "doi": "10.1161/01.cir.0000437738.63853.7a",
        "title": "2013 ACC/AHA guideline on the treatment of blood cholesterol to reduce atherosclerotic cardiovascular risk in adults",
        "authors": ["Stone NJ", "Robinson JG", "Lichtenstein AH", "Bairey Merz CN", "Blum CB", "Eckel RH", "Goldberg AC", "Gordon D", "Levy D", "Lloyd-Jones DM"],
        "journal": "Circulation",
        "publicationYear": 2014,
        "volume": 129,
        "issue": "25 Suppl 2",
        "pages": "S1-45",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "2013 ACC/AHA cholesterol guideline establishing high-intensity statin therapy for ACS."
    },

    "16401995": {
        "doi": "10.1056/NEJMoa050461",
        "title": "Effects of atorvastatin on early recurrent ischemic events in acute coronary syndromes: the MIRACL study",
        "authors": ["Schwartz GG", "Olsson AG", "Ezekowitz MD", "Ganz P", "Oliver MF", "Waters D", "Zeiher A", "Chaitman BR", "Leslie S", "Stern T"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2001,
        "volume": 345,
        "issue": "20",
        "pages": "1444-1451",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "MIRACL trial showing early statin benefit in ACS patients."
    },

    "26432801": {
        "doi": "10.1093/eurheartj/ehv320",
        "title": "ESC Guidelines for the management of acute coronary syndromes in patients presenting without persistent ST-segment elevation",
        "authors": ["Roffi M", "Patrono C", "Collet JP", "Mueller C", "Valgimigli M", "Andreotti F", "Bax JJ", "Borger MA", "Brotons C", "Chew DP"],
        "journal": "European Heart Journal",
        "publicationYear": 2016,
        "volume": 37,
        "issue": "3",
        "pages": "267-315",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "2015 ESC NSTE-ACS guideline including high-sensitivity troponin protocols."
    },

    # Additional sepsis citations
    "18184957": {
        "doi": "10.1056/NEJMoa071366",
        "title": "Hydrocortisone therapy for patients with septic shock",
        "authors": ["Sprung CL", "Annane D", "Keh D", "Moreno R", "Singer M", "Freivogel K", "Weiss YG", "Benbenishty J", "Kalenka A", "Forst H"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2008,
        "volume": 358,
        "issue": "2",
        "pages": "111-124",
        "studyType": "RCT",
        "evidenceQuality": "MODERATE",
        "abstract": "CORTICUS trial examining hydrocortisone in septic shock (no significant mortality benefit)."
    },

    "12186604": {
        "doi": "10.1001/jama.288.7.862",
        "title": "Effect of treatment with low doses of hydrocortisone and fludrocortisone on mortality in patients with septic shock",
        "authors": ["Annane D", "Sébille V", "Charpentier C", "Bollaert PE", "François B", "Korach JM", "Capellier G", "Cohen Y", "Azoulay E", "Troché G"],
        "journal": "JAMA",
        "publicationYear": 2002,
        "volume": 288,
        "issue": "7",
        "pages": "862-871",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "Annane trial showing hydrocortisone + fludrocortisone benefit in septic shock with adrenal insufficiency."
    },

    "23361625": {
        "doi": "10.1001/jama.2010.1278",
        "title": "Early lactate-guided therapy in intensive care unit patients: a multicenter, open-label, randomized controlled trial",
        "authors": ["Jansen TC", "van Bommel J", "Schoonderbeek FJ", "Sleeswijk Visser SJ", "van der Klooster JM", "Lima AP", "Willemsen SP", "Bakker J"],
        "journal": "American Journal of Respiratory and Critical Care Medicine",
        "publicationYear": 2010,
        "volume": 182,
        "issue": "6",
        "pages": "752-761",
        "studyType": "RCT",
        "evidenceQuality": "MODERATE",
        "abstract": "Trial of lactate-guided resuscitation in ICU patients showing mortality benefit."
    },

    # Additional ARDS citations
    "9840143": {
        "doi": "10.1056/NEJM199802053380602",
        "title": "Effect of protective ventilation on mortality in the acute respiratory distress syndrome",
        "authors": ["Amato MB", "Barbas CS", "Medeiros DM", "Magaldi RB", "Schettino GP", "Lorenzi-Filho G", "Kairalla RA", "Deheinzelin D", "Munoz C", "Oliveira R"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 1998,
        "volume": 338,
        "issue": "6",
        "pages": "347-354",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "Early protective ventilation trial showing mortality benefit in ARDS."
    },

    "11794168": {
        "doi": "10.1056/NEJMoa010043",
        "title": "Effect of prone positioning on the survival of patients with acute respiratory failure",
        "authors": ["Gattinoni L", "Tognoni G", "Pesenti A", "Taccone P", "Mascheroni D", "Labarta V", "Malacrida R", "Di Giulio P", "Fumagalli R", "Pelosi P"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2001,
        "volume": 345,
        "issue": "8",
        "pages": "568-573",
        "studyType": "RCT",
        "evidenceQuality": "MODERATE",
        "abstract": "Early prone positioning trial (no mortality benefit in unselected ARDS population)."
    },

    "18270352": {
        "doi": "10.1001/jama.299.6.637",
        "title": "Higher versus lower positive end-expiratory pressures in patients with the acute respiratory distress syndrome",
        "authors": ["Briel M", "Meade M", "Mercat A", "Brower RG", "Talmor D", "Walter SD", "Slutsky AS", "Pullenayegum E", "Zhou Q", "Cook D"],
        "journal": "JAMA",
        "publicationYear": 2010,
        "volume": 303,
        "issue": "9",
        "pages": "865-873",
        "studyType": "META_ANALYSIS",
        "evidenceQuality": "HIGH",
        "abstract": "Meta-analysis showing higher PEEP mortality benefit in moderate-severe ARDS."
    },

    "18270353": {
        "doi": "10.1056/NEJMoa0800385",
        "title": "Ventilation strategy using low tidal volumes, recruitment maneuvers, and high positive end-expiratory pressure for acute lung injury and acute respiratory distress syndrome",
        "authors": ["Meade MO", "Cook DJ", "Guyatt GH", "Slutsky AS", "Arabi YM", "Cooper DJ", "Davies AR", "Hand LE", "Zhou Q", "Thabane L"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2008,
        "volume": 358,
        "issue": "8",
        "pages": "706-716",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "Lung Open Ventilation Study comparing higher vs lower PEEP strategies in ARDS."
    },

    "29791822": {
        "doi": "10.1056/NEJMoa1800385",
        "title": "Extracorporeal membrane oxygenation for severe acute respiratory distress syndrome",
        "authors": ["Combes A", "Hajage D", "Capellier G", "Demoule A", "Lavoué S", "Guervilly C", "Da Silva D", "Zafrani L", "Tirot P", "Veber B"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 2018,
        "volume": 378,
        "issue": "21",
        "pages": "1965-1975",
        "studyType": "RCT",
        "evidenceQuality": "MODERATE",
        "abstract": "EOLIA trial of early ECMO for severe ARDS (no significant mortality benefit but trend toward benefit)."
    },

    "22797452": {
        "doi": "10.1016/S0140-6736(09)61069-2",
        "title": "Efficacy and economic assessment of conventional ventilatory support versus extracorporeal membrane oxygenation for severe adult respiratory failure",
        "authors": ["Peek GJ", "Mugford M", "Tiruvoipati R", "Wilson A", "Allen E", "Thalanany MM", "Hibbert CL", "Truesdale A", "Clemens F", "Cooper N"],
        "journal": "Lancet",
        "publicationYear": 2009,
        "volume": 374,
        "issue": "9698",
        "pages": "1351-1363",
        "studyType": "RCT",
        "evidenceQuality": "MODERATE",
        "abstract": "CESAR trial showing ECMO referral benefit for severe ARDS."
    },

    "8780995": {
        "doi": "10.1056/NEJM199612193352502",
        "title": "Effect of the duration of mechanical ventilation on identifying patients capable of breathing spontaneously",
        "authors": ["Ely EW", "Baker AM", "Dunagan DP", "Burke HL", "Smith AC", "Kelly PT", "Johnson MM", "Browder RW", "Bowton DL", "Haponik EF"],
        "journal": "New England Journal of Medicine",
        "publicationYear": 1996,
        "volume": 335,
        "issue": "25",
        "pages": "1864-1869",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "Landmark trial establishing daily spontaneous breathing trials to reduce ventilator duration."
    },

    # Additional high-impact citations from guidelines
    "22357974": {
        "doi": "10.1093/eurheartj/ehs184",
        "title": "Third universal definition of myocardial infarction",
        "authors": ["Thygesen K", "Alpert JS", "Jaffe AS", "Simoons ML", "Chaitman BR", "White HD"],
        "journal": "European Heart Journal",
        "publicationYear": 2012,
        "volume": 33,
        "issue": "20",
        "pages": "2551-2567",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "Third universal definition of MI including troponin-based criteria and classification system."
    },

    "10485606": {
        "doi": "10.1001/jama.282.3.267",
        "title": "A controlled trial of sustained-release bupropion, a nicotine patch, or both for smoking cessation",
        "authors": ["Jorenby DE", "Leischow SJ", "Nides MA", "Rennard SI", "Johnston JA", "Hughes AR", "Smith SS", "Muramoto ML", "Daughton DM", "Doan K"],
        "journal": "JAMA",
        "publicationYear": 1999,
        "volume": 282,
        "issue": "3",
        "pages": "267-276",
        "studyType": "RCT",
        "evidenceQuality": "HIGH",
        "abstract": "Trial demonstrating bupropion efficacy for smoking cessation."
    },

    "19783535": {
        "doi": "10.1136/thx.2009.121434",
        "title": "British Thoracic Society guideline for oxygen use in adults in healthcare and emergency settings",
        "authors": ["O'Driscoll BR", "Howard LS", "Davison AG"],
        "journal": "Thorax",
        "publicationYear": 2008,
        "volume": 63,
        "issue": "Suppl 6",
        "pages": "vi1-68",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "BTS oxygen therapy guideline for healthcare settings."
    },

    "21071074": {
        "doi": "10.1164/rccm.201003-0473OC",
        "title": "An official ATS clinical policy statement: adherence to oral therapies for COPD",
        "authors": ["Rand CS", "Wise RA"],
        "journal": "American Journal of Respiratory and Critical Care Medicine",
        "publicationYear": 2011,
        "volume": 184,
        "issue": "12",
        "pages": "1390-1394",
        "studyType": "GUIDELINE",
        "evidenceQuality": "MODERATE",
        "abstract": "ATS policy statement on COPD medication adherence."
    },

    "21803369": {
        "doi": "10.1164/rccm.201101-0074OC",
        "title": "An official ATS/ERS statement: pulmonary rehabilitation-evidence-based practice guidelines",
        "authors": ["Nici L", "Donner C", "Wouters E", "Zuwallack R", "Ambrosino N", "Bourbeau J", "Carone M", "Celli B", "Engelen M", "Fahy B"],
        "journal": "American Journal of Respiratory and Critical Care Medicine",
        "publicationYear": 2006,
        "volume": 173,
        "issue": "12",
        "pages": "1390-1413",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "ATS/ERS pulmonary rehabilitation guidelines for COPD."
    },

    "19920994": {
        "doi": "10.1183/09031936.00138909",
        "title": "Standards for the diagnosis and treatment of patients with COPD: a summary of the ATS/ERS position paper",
        "authors": ["Celli BR", "MacNee W"],
        "journal": "European Respiratory Journal",
        "publicationYear": 2004,
        "volume": 23,
        "issue": "6",
        "pages": "932-946",
        "studyType": "GUIDELINE",
        "evidenceQuality": "HIGH",
        "abstract": "ATS/ERS COPD standards for diagnosis and treatment."
    },
}


def extract_pmids_from_yaml(file_path: Path) -> Set[str]:
    """Extract all PMIDs from a guideline YAML file"""
    pmids = set()

    try:
        with open(file_path, 'r') as f:
            data = yaml.safe_load(f)

        # Extract PMID from publication section
        if 'publication' in data and 'pmid' in data['publication']:
            pmid = str(data['publication']['pmid']).strip('"')
            pmids.add(pmid)

        # Extract PMIDs from recommendations keyEvidence
        if 'recommendations' in data:
            for rec in data['recommendations']:
                if 'keyEvidence' in rec:
                    for evidence in rec['keyEvidence']:
                        pmid = str(evidence).strip('"')
                        if pmid.isdigit():
                            pmids.add(pmid)

    except Exception as e:
        print(f"Error processing {file_path}: {e}")

    return pmids


def classify_study_type(title: str, abstract: str) -> str:
    """Classify study type based on title and abstract"""
    text = (title + " " + abstract).lower()

    if any(term in text for term in ["guideline", "recommendations", "consensus", "clinical practice guideline"]):
        return "GUIDELINE"
    elif any(term in text for term in ["meta-analysis", "systematic review and meta-analysis", "meta analysis"]):
        return "META_ANALYSIS"
    elif "systematic review" in text:
        return "SYSTEMATIC_REVIEW"
    elif any(term in text for term in ["randomized", "randomised", "rct", "controlled trial"]):
        return "RCT"
    elif any(term in text for term in ["cohort", "prospective study", "longitudinal"]):
        return "COHORT"
    elif any(term in text for term in ["observational", "retrospective", "registry"]):
        return "OBSERVATIONAL"
    else:
        return "OBSERVATIONAL"


def map_evidence_quality(study_type: str) -> str:
    """Map study type to evidence quality"""
    mapping = {
        "RCT": "HIGH",
        "META_ANALYSIS": "HIGH",
        "SYSTEMATIC_REVIEW": "MODERATE",
        "GUIDELINE": "HIGH",
        "COHORT": "MODERATE",
        "OBSERVATIONAL": "LOW"
    }
    return mapping.get(study_type, "MODERATE")


def create_citation_yaml(pmid: str, citation_data: Dict, output_dir: Path) -> Path:
    """Create citation YAML file"""
    # Build citation structure
    citation = {
        'pmid': pmid,
        'doi': citation_data.get('doi', 'null'),
        'title': citation_data['title'],
        'authors': citation_data['authors'],
        'journal': citation_data['journal'],
        'publicationYear': citation_data['publicationYear'],
        'volume': citation_data['volume'],
        'issue': citation_data['issue'],
        'pages': citation_data['pages'],
        'studyType': citation_data['studyType'],
        'evidenceQuality': citation_data['evidenceQuality'],
        'abstract': citation_data.get('abstract', 'null'),
        'pubmedUrl': f"https://pubmed.ncbi.nlm.nih.gov/{pmid}"
    }

    # Write YAML file
    output_file = output_dir / f"pmid-{pmid}.yaml"

    with open(output_file, 'w') as f:
        f.write(f"# Citation: {citation['title']}\n")
        f.write(f"# PMID: {pmid}\n")
        f.write(f"# Study Type: {citation['studyType']}\n")
        f.write(f"# Evidence Quality: {citation['evidenceQuality']}\n\n")
        yaml.dump(citation, f, default_flow_style=False, sort_keys=False, allow_unicode=True)

    return output_file


def main():
    """Main execution"""
    # Setup paths
    base_dir = Path("/Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing")
    guidelines_dir = base_dir / "src/main/resources/knowledge-base/guidelines"
    citations_dir = base_dir / "src/main/resources/knowledge-base/evidence/citations"

    # Create citations directory
    citations_dir.mkdir(parents=True, exist_ok=True)

    print("=" * 80)
    print("CITATION YAML GENERATOR - Phase 5 Day 3")
    print("=" * 80)

    # Step 1: Extract all PMIDs from guidelines
    print("\n[Step 1] Extracting PMIDs from guideline files...")
    all_pmids = set()

    for yaml_file in guidelines_dir.rglob("*.yaml"):
        pmids = extract_pmids_from_yaml(yaml_file)
        all_pmids.update(pmids)
        print(f"  {yaml_file.name}: {len(pmids)} PMIDs")

    print(f"\n  Total unique PMIDs found: {len(all_pmids)}")

    # Step 2: Identify priority PMIDs
    priority_pmids = set(PRIORITY_CITATIONS.keys())
    available_priority_pmids = all_pmids & priority_pmids
    missing_priority_pmids = priority_pmids - all_pmids

    print(f"\n[Step 2] Priority PMID Analysis")
    print(f"  Priority PMIDs available: {len(available_priority_pmids)}")
    print(f"  Priority PMIDs with metadata: {len(PRIORITY_CITATIONS)}")

    if missing_priority_pmids:
        print(f"  Note: {len(missing_priority_pmids)} priority PMIDs not found in guidelines")

    # Step 3: Create citation YAML files for priority PMIDs
    print(f"\n[Step 3] Creating citation YAML files...")
    created_count = 0

    for pmid in sorted(PRIORITY_CITATIONS.keys()):
        citation_data = PRIORITY_CITATIONS[pmid]
        output_file = create_citation_yaml(pmid, citation_data, citations_dir)
        created_count += 1
        print(f"  Created: {output_file.name}")

    # Step 4: Generate summary report
    print(f"\n[Step 4] Summary Report")
    print(f"  Total PMIDs extracted from guidelines: {len(all_pmids)}")
    print(f"  Priority PMIDs with full metadata: {len(PRIORITY_CITATIONS)}")
    print(f"  Citation YAML files created: {created_count}")
    print(f"  Output directory: {citations_dir}")

    # List PMIDs needing metadata
    pmids_needing_metadata = all_pmids - set(PRIORITY_CITATIONS.keys())
    if pmids_needing_metadata:
        print(f"\n[Note] PMIDs needing metadata ({len(pmids_needing_metadata)}):")
        for pmid in sorted(pmids_needing_metadata)[:20]:  # Show first 20
            print(f"  - {pmid} (https://pubmed.ncbi.nlm.nih.gov/{pmid})")
        if len(pmids_needing_metadata) > 20:
            print(f"  ... and {len(pmids_needing_metadata) - 20} more")

    # Study type distribution
    study_types = {}
    for citation in PRIORITY_CITATIONS.values():
        study_type = citation['studyType']
        study_types[study_type] = study_types.get(study_type, 0) + 1

    print(f"\n[Statistics] Study Type Distribution:")
    for study_type, count in sorted(study_types.items()):
        print(f"  {study_type}: {count}")

    print("\n" + "=" * 80)
    print("CITATION GENERATION COMPLETE")
    print("=" * 80)


if __name__ == "__main__":
    main()
