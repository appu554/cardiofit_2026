# 🔐 Manual Neo4j Password Change - Quick Guide

## ⚡ Quick Command (Copy & Paste)

**Run this in your terminal** (takes 15 seconds):

```bash
docker exec -it neo4j cypher-shell -u neo4j -p neo4j
```

**At the `neo4j>` prompt, paste this**:

```cypher
ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';
:exit
```

---

## 📋 Step-by-Step

### Step 1: Connect to Neo4j Shell
```bash
docker exec -it neo4j cypher-shell -u neo4j -p neo4j
```

**You'll see**:
```
Password change required
neo4j>
```

### Step 2: Change Password
**Copy and paste this exact command**:
```cypher
ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';
```

**Press Enter**. You'll see:
```
0 rows available after 123 ms, consumed after another 0 ms
```

### Step 3: Exit
```cypher
:exit
```

---

## ✅ Verify It Worked

**Run this command**:
```bash
echo "RETURN 'SUCCESS!' AS status;" | docker exec -i neo4j cypher-shell -u neo4j -p 'CardioFit2024!'
```

**Expected output**:
```
+------------+
| status     |
+------------+
| "SUCCESS!" |
+------------+
```

✅ **If you see "SUCCESS!"** - You're done! Module 2 is ready.

❌ **If you see "Access denied"** - The password wasn't changed. Try again from Step 1.

---

## 🚀 What Happens Next

Once the password is changed, Module 2 will be able to:

1. **Query care network data** from Neo4j when processing first-time patients
2. **Populate patient snapshots** with:
   - Care team (providers assigned to patient)
   - Risk cohorts (patient belongs to which risk groups)
   - Care pathways (active clinical pathways)
   - Related patients (family/care network connections)

3. **Update care networks** when encounters close
4. **Gracefully degrade** if Neo4j is temporarily unavailable (continues processing without graph data)

---

## 🔍 Test Neo4j Connection from Browser (Optional)

1. Open: http://localhost:55001
2. Connect URL: `bolt://localhost:55002`
3. Username: `neo4j`
4. Password: `CardioFit2024!`

You should see the Neo4j Browser interface.

---

## 📚 Next Steps After Password Change

1. ✅ Complete the manual password change above
2. Run verification command to confirm
3. Optionally load sample graph data (see `NEO4J_SETUP_FOR_MODULE2.md`)
4. Proceed with Module 2 Phase 5 testing

---

## 🐛 Troubleshooting

**Problem**: `docker exec -it` says "the input device is not a TTY"

**Solution**: Make sure you're running this in a **real terminal** (not through a script or automation). Terminal, iTerm2, or any shell window will work.

**Problem**: Still getting "credentials expired" after changing password

**Solution**:
1. Restart Neo4j: `docker restart neo4j`
2. Wait 10 seconds
3. Try the verification command again

**Problem**: Can't connect to Neo4j at all

**Solution**: Verify Neo4j is running: `docker ps | grep neo4j`

---

## 🎯 Summary

**What you need to do**:
1. Run: `docker exec -it neo4j cypher-shell -u neo4j -p neo4j`
2. Paste: `ALTER CURRENT USER SET PASSWORD FROM 'neo4j' TO 'CardioFit2024!';`
3. Exit: `:exit`
4. Verify: `echo "RETURN 'SUCCESS!' AS status;" | docker exec -i neo4j cypher-shell -u neo4j -p 'CardioFit2024!'`

**Time required**: ~15 seconds

**What this enables**: Module 2 Neo4j integration for care network queries

**Status after completion**: Ready for Phase 5 testing! 🚀
