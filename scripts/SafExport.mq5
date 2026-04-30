//+------------------------------------------------------------------+
//| SafExport.mq5 — Set & Trend live H4 OHLCV exporter                |
//|                                                                   |
//| Runs on any chart in MetaTrader 5. On every H4 close (plus a 90s  |
//| grace window so the close print is final), iterates the symbol    |
//| list and POSTs the most recent N closed H4 bars per symbol to a   |
//| companion Go listener (cmd/mt5-listen) on localhost:9991.         |
//|                                                                   |
//| Companion piece: backend/cmd/mt5-listen/main.go (validates with   |
//| sanity bands, upserts to candles_h4, fires SignalEvaluator).      |
//|                                                                   |
//| Setup:                                                            |
//|   1. Open MetaEditor, copy this file into MQL5/Experts/.          |
//|   2. Press F7 to compile (no errors expected).                    |
//|   3. In MT5: Tools → Options → Expert Advisors. Tick "Allow       |
//|      WebRequest" and add http://localhost:9991 to the URL list.   |
//|   4. Drag SafExport from Navigator onto any chart (any symbol,    |
//|      any timeframe — symbol selection is set in EA inputs).       |
//|   5. Tick "Allow Algo Trading" in the toolbar.                    |
//|                                                                   |
//| The EA does NOT trade. It only reads bars and posts JSON.         |
//+------------------------------------------------------------------+
#property copyright "Set & Trend"
#property version   "1.00"
#property strict

input string  InpListenerURL = "http://192.168.1.2:9991/ingest"; // Go listener endpoint (Wine loopback is blocked; use the host LAN IP)
input string  InpSymbols     = "AUDCHF,AUDUSD,CADJPY,CHFJPY,EURAUD,EURCAD,EURCHF,EURGBP,EURJPY,EURUSD,GBPCHF,GBPJPY,GBPUSD,NZDJPY,NZDUSD,USDCAD,USDCHF,USDJPY,XAUUSD"; // 19 SAF pairs (DB form)
input int     InpBarsPerCycle = 10;     // closed H4 bars to send per symbol
input int     InpGraceSec      = 90;    // delay after H4 close before posting
input bool    InpDottedTickers = true;  // broker uses XXX.YYY format? (MetaQuotes-Demo: yes)
input string  InpBrokerSuffix  = "";    // additional suffix (e.g. ".m", "-Pro") applied AFTER dot-injection
input bool    InpVerbose       = true;

int     g_lastH4Ticket = -1;       // lastH4StartUnix posted (avoids double-post)
string  g_symbols[];               // parsed symbol list (DB form)
string  g_brokerMap[];             // parallel — broker ticker per db symbol ("" = unresolvable)

//+------------------------------------------------------------------+
int OnInit()
{
   StringSplit(InpSymbols, ',', g_symbols);
   for(int i=0; i<ArraySize(g_symbols); i++)
   {
      StringTrimRight(g_symbols[i]);
      StringTrimLeft(g_symbols[i]);
   }
   BuildBrokerMap();
   PrintFormat("SafExport ready — %d symbols, listener=%s", ArraySize(g_symbols), InpListenerURL);
   EventSetTimer(15);   // poll every 15s; cheap, no broker traffic
   return(INIT_SUCCEEDED);
}

//+------------------------------------------------------------------+
//| Resolve every DB symbol to the broker's actual ticker by trying  |
//| common variants and (as a fallback) scanning SymbolsTotal for a  |
//| substring match. Result cached in g_brokerMap.                   |
//+------------------------------------------------------------------+
void BuildBrokerMap()
{
   int n = ArraySize(g_symbols);
   ArrayResize(g_brokerMap, n);
   int resolved = 0;
   for(int i=0; i<n; i++)
   {
      string db = g_symbols[i];
      g_brokerMap[i] = "";
      if(db == "") continue;

      // Build candidate list, ordered by likelihood:
      string cands[];
      int cn = 0;
      ArrayResize(cands, 8);
      cands[cn++] = db;                                                                                 // EURUSD
      if(InpBrokerSuffix != "") cands[cn++] = db + InpBrokerSuffix;                                    // EURUSD.m
      if(StringLen(db) == 6)
      {
         string dotted = StringSubstr(db,0,3) + "." + StringSubstr(db,3,3);                            // EUR.USD
         cands[cn++] = dotted;
         if(InpBrokerSuffix != "") cands[cn++] = dotted + InpBrokerSuffix;                             // EUR.USD.m
      }
      ArrayResize(cands, cn);

      for(int c=0; c<cn; c++)
      {
         if(SymbolSelect(cands[c], true))
         {
            g_brokerMap[i] = cands[c];
            break;
         }
      }
      // Fallback: scan all available broker symbols for one whose first
      // 6 alphanumeric chars match the DB symbol (handles weird formats
      // like "EUR_USD", "EURUSD#", etc.).
      if(g_brokerMap[i] == "")
      {
         int total = SymbolsTotal(false);
         for(int s=0; s<total; s++)
         {
            string bs = SymbolName(s, false);
            string clean = "";
            for(int k=0; k<StringLen(bs) && StringLen(clean)<6; k++)
            {
               ushort ch = StringGetCharacter(bs, k);
               if((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')) clean += ShortToString(ch);
            }
            if(StringCompare(clean, db, false) == 0)
            {
               if(SymbolSelect(bs, true))
               {
                  g_brokerMap[i] = bs;
                  break;
               }
            }
         }
      }

      if(g_brokerMap[i] == "")
         PrintFormat("[%s] no broker ticker found (tried %d variants)", db, cn);
      else
      {
         PrintFormat("[%s] -> %s", db, g_brokerMap[i]);
         resolved++;
      }
   }
   PrintFormat("BrokerMap: resolved %d/%d symbols", resolved, n);
}

void OnDeinit(const int reason) { EventKillTimer(); }

//+------------------------------------------------------------------+
//| Timer fires every 15s. We post once per H4 boundary.              |
//+------------------------------------------------------------------+
void OnTimer()
{
   datetime now = TimeCurrent();
   datetime h4Start = now - (now % (4*3600));      // current H4 bar's open time
   datetime lastClosedH4 = h4Start;                 // it's also the last closed bar's open time...
                                                    // wait — lastClosedH4 is the bar BEFORE current, so:
   datetime lastClosedH4Start = h4Start - 4*3600;
   datetime lastClosedH4Close = h4Start;            // close == start of the next bar

   // Only post once per closed H4 bar, after the grace window.
   if((int)lastClosedH4Start == g_lastH4Ticket)
      return;
   if(now < lastClosedH4Close + InpGraceSec)
      return;

   if(InpVerbose) PrintFormat("Cycle: lastClosedH4Start=%s", TimeToString(lastClosedH4Start, TIME_DATE|TIME_MINUTES));

   int posted = 0;
   for(int i=0; i<ArraySize(g_symbols); i++)
   {
      string dbSym = g_symbols[i];
      if(dbSym == "") continue;
      string brokerSym = g_brokerMap[i];
      if(brokerSym == "")
      {
         // Already logged at OnInit; just skip silently per cycle.
         continue;
      }
      if(PostSymbol(dbSym, brokerSym, InpBarsPerCycle))
         posted++;
   }
   PrintFormat("Cycle done: %d/%d symbols posted", posted, ArraySize(g_symbols));
   g_lastH4Ticket = (int)lastClosedH4Start;
}

//+------------------------------------------------------------------+
//| Build JSON for one symbol's last N closed H4 bars and POST it.   |
//+------------------------------------------------------------------+
bool PostSymbol(const string dbSym, const string brokerSym, const int n)
{
   MqlRates rates[];
   ArraySetAsSeries(rates, true);
   int copied = CopyRates(brokerSym, PERIOD_H4, 1, n, rates);   // start=1 → skip the still-forming current bar
   if(copied <= 0)
   {
      PrintFormat("[%s] CopyRates failed (err=%d)", brokerSym, GetLastError());
      return false;
   }

   string body = "{\"symbol\":\"" + dbSym + "\",\"bars\":[";
   for(int i=copied-1; i>=0; i--)   // oldest → newest in payload
   {
      string sep = (i == copied-1 ? "" : ",");
      body += sep + StringFormat(
         "{\"time\":%d,\"open\":%.5f,\"high\":%.5f,\"low\":%.5f,\"close\":%.5f,\"volume\":%d}",
         (long)rates[i].time, rates[i].open, rates[i].high, rates[i].low, rates[i].close, (int)rates[i].tick_volume
      );
   }
   body += "]}";

   char post[];
   StringToCharArray(body, post, 0, StringLen(body));
   char result[];
   string headers = "Content-Type: application/json\r\n";
   string respHeaders;

   ResetLastError();
   int code = WebRequest("POST", InpListenerURL, headers, 5000, post, result, respHeaders);
   if(code != 200)
   {
      string respBody = CharArrayToString(result);
      PrintFormat("[%s] POST failed: http=%d err=%d resp=%s", dbSym, code, GetLastError(), StringSubstr(respBody, 0, 200));
      return false;
   }
   if(InpVerbose) PrintFormat("[%s] posted %d bars", dbSym, copied);
   return true;
}
//+------------------------------------------------------------------+
