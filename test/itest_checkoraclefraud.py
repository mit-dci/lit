
def run_test(env):

    lit = env.lits[0]

    s1string = "424b62134ec5ff7f8ff25de43917a03582e253283575b8f815d26bbdc27d17f8"
    h1string = "9bd6e409476804596c2793eb722fd23479f2a1a8a439e8cb47faed68dc660535"
    s2string = "4a61ef074997f0f0039c71e9dd91d15263c6f98bc54034336491d5e8a5445f4c"
    h2string = "dc2b6ce71bb4099ca53c70eadcd1d9d4be46b65c1e0b540528e619fd236ae09a"

    rpoint = "02f8460e855b091cec11ccf4a85064d4a8a7d3a2970b957a2165564b537d510bb4"  
    apoint = "029bc17aed9a0a5821b5b0425d8260d66f0529eb357a0b036765d68904152f618a"

    res = lit.rpc.DifferentResultsFraud(Sfirst=s1string, Hfirst=h1string, Ssecond=s2string, Hsecond=h2string, Rpoint=rpoint, Apoint=apoint)
    
    print('Whether the oracle publish two different results or not.')
    print(res)

